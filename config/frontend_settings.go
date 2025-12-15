package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const frontendSettingsFile = "frontend_settings.json"

type FrontendGpsLabel struct {
	Label    string `json:"label"`
	Location string `json:"location"`
}

type FrontendSettings struct {
	DefaultEmail string             `json:"defaultEmail"`
	GpsLabels    []FrontendGpsLabel `json:"gpsLabels"`
}

var frontendMu sync.Mutex

func frontendSettingsPath() string {
	return filepath.Join(overrideDir, frontendSettingsFile)
}

func GetFrontendSettings() (FrontendSettings, error) {
	frontendMu.Lock()
	defer frontendMu.Unlock()

	path := frontendSettingsPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FrontendSettings{DefaultEmail: "", GpsLabels: []FrontendGpsLabel{}}, nil
		}
		return FrontendSettings{}, err
	}
	var s FrontendSettings
	if err := json.Unmarshal(b, &s); err != nil {
		return FrontendSettings{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.GpsLabels == nil {
		s.GpsLabels = []FrontendGpsLabel{}
	}
	return s, nil
}

func UpdateFrontendSettings(s FrontendSettings) (FrontendSettings, error) {
	frontendMu.Lock()
	defer frontendMu.Unlock()

	if s.GpsLabels == nil {
		s.GpsLabels = []FrontendGpsLabel{}
	}

	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		return FrontendSettings{}, fmt.Errorf("mkdir %s: %w", overrideDir, err)
	}
	path := frontendSettingsPath()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return FrontendSettings{}, fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return FrontendSettings{}, fmt.Errorf("write %s: %w", path, err)
	}
	return s, nil
}
