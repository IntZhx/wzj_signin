# 微助教签到助手

一个运行在本机/服务器上的小工具：通过 Web 页面管理 OpenID 监控，自动轮询可签到列表。

本项目基于上游仓库演进（上游地址：https://github.com/Azuka753/wzj_sign_public）。

## 免责声明

- 本项目与“微助教 / teachermate”无任何关联。
- 请在遵守所在组织/课程/平台规则的前提下使用。

## 功能

- 自动轮询 OpenID 的可签到列表（按 `app.interval` 秒）
- 普通签到 / GPS 签到 / 二维码签到提醒
- 二维码签到：触发后可发送邮件，并提供二维码页面（自动更新二维码）
- Web 页面：`/home`、`/submit`、`/history`、`/settings`
- 本机数据持久化（默认在 `data/` 目录）

## 运行前准备

- Go 1.20+（本机运行时需要）
- Redis（本机或局域网均可）

## 快速开始（本机运行，Windows）

1) 启动 Redis

- 默认读取 `localhost:6379`（可用环境变量 `REDIS_ADDRESS` 覆盖）

2) 启动服务

```powershell
go run .
```

3) 打开页面

- http://localhost:8080/home

## Docker 运行

### 方式一：docker-compose（自带 Redis）

```bash
docker compose up -d --build
```

启动后：

- 应用：http://localhost:18080/home
- Redis：仅供容器内部使用（地址 `redis:6379`，默认不再映射到宿主机端口）

数据持久化：默认使用 `./data` 目录作为绑定挂载。首次运行时 `data/*.json` 可能不存在，保存设置（例如新增 GPS 标签）时会自动创建。


### 方式二：纯 Docker（使用外部 Redis）

```bash
docker build -t wzj-sign:local .

docker run -d \
  --name wzj_sign \
  -p 18080:8080 \
  -e PORT=8080 \
  -e SERVER_ADDRESS=http://localhost:18080 \
  -e REDIS_ADDRESS=host.docker.internal:6379 \
  -v %CD%/data:/app/data \
  wzj-sign:local
```

如需使用 `config.yml`：可额外挂载 `-v %CD%/local/config.yml:/app/config.yml:ro`（也可以不提供，直接使用默认值 + 设置页）。


## 配置与数据（重要）

### 1) 非敏感配置：config.yml（可选）

不提供 `config.yml` 也能启动（使用默认值）。

推荐把真实配置放在：`local/config.yml`（已被 `.gitignore` 忽略，适合放个人/本机配置）。


模板文件：`examples/config.example.yml`（复制一份到 `local/config.yml` 再修改）。

首次运行时如果检测到你还没有任何 `config.yml`，程序会尝试自动从 `examples/config.example.yml` 生成一份 `local/config.yml` 作为起点（不会覆盖你已有的文件）。

### 2) 敏感配置：data/secrets.json（本机）

敏感字段（如 Redis 密码、邮箱 SMTP 密码）不要放在 `config.yml`。

创建文件：`data/secrets.json`（模板：`examples/secrets.example.json`）。

说明：

- 启动时会优先读取 `data/secrets.json` 覆盖密码类配置
- 设置页更新邮箱密码时，也只会写入 `data/secrets.json`（不会回显密码）

### 3) 本机持久化文件（默认在 data/）

- `data/appconfig.json`：UI 配置（不含密码）
- `data/frontend_settings.json`：默认邮箱、GPS 标签等
- `data/secrets.json`：敏感信息（密码类）

首次运行时这些 `data/*.json` 可能不存在，程序会自动创建（不会覆盖已有内容）。

## Web 页面说明

- `/settings`：保存默认邮箱、管理 GPS 标签、配置邮件发送与拟真延迟
- `/submit`：粘贴 OpenID 或包含 openid 的链接，选择 GPS 标签并提交
- `/home`：运行概览与使用说明
- `/history`：查看本机记录，开始/停止轮询，可再次打开二维码页面

## 二维码签到（常见坑）

- 二维码签到与 GPS 签到互不影响。
- 当检测到二维码签到时：
  - 可发送邮件，邮件链接通常形如：`/static/qr.html?sign=...&course=...`
  - 二维码页会轮询后端接口获取二维码并自动刷新
- 扫码后 OpenID 可能会立刻失效；如需继续监控通常需要重新获取新的 OpenID

## 常见问题（Windows）

### 1) 端口被占用

如果 8080 被占用：

```powershell
$env:PORT=8081
$env:SERVER_ADDRESS="http://localhost:8081"
go run .
```

也可以使用脚本：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-local.ps1 -Port 8081
```

### 2) config.yml 编码问题

请确保 `config.yml` 为 UTF-8 编码，否则可能出现 `invalid trailing UTF-8 octet`。

> 提示：Windows 上常见端口冲突（例如 Steam 占用 8080、Memurai 占用 6379）。如果你坚持使用 8080/6379，请先关闭对应程序或修改端口映射。

## 致谢

- 上游项目：https://github.com/Azuka753/wzj_sign_public


