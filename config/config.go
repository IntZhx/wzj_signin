package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

const overrideDir = "data"
const overrideFile = "appconfig.json"
const secretsFile = "secrets.json"
const localDir = "local"

type AppConfig struct {
	NormalDelay int        `json:"normal_delay"`
	Mail        MailConfig `json:"mail"`
}

type MailConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type AppConfigForUI struct {
	NormalDelay int        `json:"normal_delay"`
	Mail        MailConfig `json:"mail"`
	PasswordSet bool       `json:"passwordSet"`
}

type Secrets struct {
	RedisPassword string `json:"redisPassword"`
	MailPassword  string `json:"mailPassword"`
}

var (
	loadOnce sync.Once
	loadErr  error
	mu       sync.Mutex
)

func Load() error {
	loadOnce.Do(func() {
		// Defaults (so config.yml can be omitted)
		viper.SetDefault("redis.address", "localhost:6379")
		viper.SetDefault("redis.password", "")
		viper.SetDefault("redis.db", 0)
		viper.SetDefault("app.interval", 8)
		viper.SetDefault("app.normal_delay", 20)
		viper.SetDefault("app.url", "http://localhost:8080")
		viper.SetDefault("mail.enabled", false)
		viper.SetDefault("mail.host", "")
		viper.SetDefault("mail.port", 0)
		viper.SetDefault("mail.username", "")
		viper.SetDefault("mail.password", "")
		viper.SetDefault("mail.from", "")

		// First-run bootstrap (so a fresh clone can save settings immediately)
		if err := ensureLocalFiles(); err != nil {
			loadErr = err
			return
		}

		// Search order: local (untracked) -> repo root -> ./config
		viper.AddConfigPath("./local")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		if err := viper.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) {
				loadErr = fmt.Errorf("read config.yml: %w", err)
				return
			}
			// config.yml not found: continue with defaults
		}

		// Apply overrides from disk (if any)
		ovr, err := readOverrides()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// continue
			} else {
				loadErr = err
				return
			}
		} else {
			applyToViper(ovr)
		}

		// Load secrets (local only). Secrets override config.yml / overrides.
		secrets, sErr := readSecrets()
		if sErr != nil {
			if !errors.Is(sErr, os.ErrNotExist) {
				loadErr = sErr
				return
			}
			secrets = Secrets{}
		}
		applySecretsToViper(secrets)

		// Migration: move sensitive fields out of config.yml / legacy appconfig.json
		migratedSecrets := false
		if strings.TrimSpace(secrets.MailPassword) == "" {
			// 1) legacy: appconfig.json used to store password
			if strings.TrimSpace(ovr.Mail.Password) != "" {
				secrets.MailPassword = strings.TrimSpace(ovr.Mail.Password)
				migratedSecrets = true
				// rewrite overrides without password
				ovr.Mail.Password = ""
				_ = writeOverrides(ovr)
			} else if p := strings.TrimSpace(viper.GetString("mail.password")); p != "" {
				// 2) from config.yml
				secrets.MailPassword = p
				migratedSecrets = true
			}
		}
		if strings.TrimSpace(secrets.RedisPassword) == "" {
			if p := strings.TrimSpace(viper.GetString("redis.password")); p != "" {
				secrets.RedisPassword = p
				migratedSecrets = true
			}
		}
		if migratedSecrets {
			_ = writeSecrets(secrets)
			applySecretsToViper(secrets)
		}
	})
	return loadErr
}

func ensureLocalFiles() error {
	// Ensure directories exist.
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", overrideDir, err)
	}
	// local/ is optional; create it to make onboarding smoother.
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", localDir, err)
	}

	// Ensure default JSON files exist (do not overwrite).
	defaultOverrides := AppConfig{
		NormalDelay: 20,
		Mail: MailConfig{
			Enabled:  false,
			Host:     "",
			Port:     0,
			Username: "",
			Password: "", // never used; kept empty
			From:     "",
		},
	}
	defaultFrontend := FrontendSettings{DefaultEmail: "", GpsLabels: []FrontendGpsLabel{}}
	defaultSecrets := Secrets{RedisPassword: "", MailPassword: ""}

	if err := ensureJSONFileIfMissing(overridesPath(), defaultOverrides); err != nil {
		return err
	}
	if err := ensureJSONFileIfMissing(frontendSettingsPath(), defaultFrontend); err != nil {
		return err
	}
	if err := ensureJSONFileIfMissing(secretsPath(), defaultSecrets); err != nil {
		return err
	}

	// Optional convenience: if this is a fresh clone and no config exists yet,
	// seed local/config.yml from examples/config.example.yml (local/ is gitignored).
	if err := maybeSeedLocalConfig(); err != nil {
		return err
	}
	return nil
}

func maybeSeedLocalConfig() error {
	// Do nothing if any config file already exists.
	_, localErr := os.Stat(filepath.Join(localDir, "config.yml"))
	_, rootErr := os.Stat("config.yml")
	_, cfgDirErr := os.Stat(filepath.Join("config", "config.yml"))
	if localErr == nil || rootErr == nil || cfgDirErr == nil {
		return nil
	}
	if !errors.Is(localErr, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", filepath.Join(localDir, "config.yml"), localErr)
	}
	if !errors.Is(rootErr, os.ErrNotExist) {
		return fmt.Errorf("stat config.yml: %w", rootErr)
	}
	if !errors.Is(cfgDirErr, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", filepath.Join("config", "config.yml"), cfgDirErr)
	}

	examplePath := filepath.Join("examples", "config.example.yml")
	b, err := os.ReadFile(examplePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", examplePath, err)
	}

	target := filepath.Join(localDir, "config.yml")
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("create %s: %w", target, err)
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

func ensureJSONFileIfMissing(path string, v any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	b = append(b, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func GetForUI() (AppConfigForUI, error) {
	if err := Load(); err != nil {
		return AppConfigForUI{}, err
	}
	cfg := AppConfigForUI{
		NormalDelay: viper.GetInt("app.normal_delay"),
		Mail: MailConfig{
			Enabled:  viper.GetBool("mail.enabled"),
			Host:     viper.GetString("mail.host"),
			Port:     viper.GetInt("mail.port"),
			Username: viper.GetString("mail.username"),
			Password: "", // never echo
			From:     viper.GetString("mail.from"),
		},
		PasswordSet: viper.GetString("mail.password") != "",
	}
	return cfg, nil
}

// UpdateFromUI persists overrides locally and applies them to viper at runtime.
// If payload.Mail.Password is empty, it will keep the previously stored password.
func UpdateFromUI(payload AppConfig) (AppConfigForUI, error) {
	if err := Load(); err != nil {
		return AppConfigForUI{}, err
	}
	mu.Lock()
	defer mu.Unlock()

	secrets, sErr := readSecrets()
	if sErr != nil {
		if !errors.Is(sErr, os.ErrNotExist) {
			return AppConfigForUI{}, sErr
		}
		secrets = Secrets{}
	}

	// Password is stored in secrets.json.
	// If payload.Mail.Password is empty, keep existing secret.
	if strings.TrimSpace(payload.Mail.Password) != "" {
		secrets.MailPassword = strings.TrimSpace(payload.Mail.Password)
		if err := writeSecrets(secrets); err != nil {
			return AppConfigForUI{}, err
		}
		applySecretsToViper(secrets)
	}

	// Never persist password into appconfig.json
	payload.Mail.Password = ""

	// Persist
	if err := writeOverrides(payload); err != nil {
		return AppConfigForUI{}, err
	}

	// Apply to viper runtime
	applyToViper(payload)

	return GetForUI()
}

func applyToViper(cfg AppConfig) {
	if cfg.NormalDelay > 0 {
		viper.Set("app.normal_delay", cfg.NormalDelay)
	}
	viper.Set("mail.enabled", cfg.Mail.Enabled)
	if cfg.Mail.Host != "" {
		viper.Set("mail.host", cfg.Mail.Host)
	}
	if cfg.Mail.Port != 0 {
		viper.Set("mail.port", cfg.Mail.Port)
	}
	if cfg.Mail.Username != "" {
		viper.Set("mail.username", cfg.Mail.Username)
	}
	// password is stored in secrets.json, not in overrides
	if cfg.Mail.From != "" {
		viper.Set("mail.from", cfg.Mail.From)
	}
}

func overridesPath() string {
	return filepath.Join(overrideDir, overrideFile)
}

func secretsPath() string {
	return filepath.Join(overrideDir, secretsFile)
}

func readSecrets() (Secrets, error) {
	path := secretsPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return Secrets{}, err
	}
	var s Secrets
	if err := json.Unmarshal(b, &s); err != nil {
		return Secrets{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}

func writeSecrets(s Secrets) error {
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", overrideDir, err)
	}
	path := secretsPath()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func applySecretsToViper(s Secrets) {
	if strings.TrimSpace(s.RedisPassword) != "" {
		viper.Set("redis.password", strings.TrimSpace(s.RedisPassword))
	}
	if strings.TrimSpace(s.MailPassword) != "" {
		viper.Set("mail.password", strings.TrimSpace(s.MailPassword))
	}
}

func readOverrides() (AppConfig, error) {
	path := overridesPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func writeOverrides(cfg AppConfig) error {
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", overrideDir, err)
	}
	path := overridesPath()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal overrides: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
