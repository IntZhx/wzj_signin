param([int]$Port = 8080)
$proj = (Resolve-Path (Join-Path "$PSScriptRoot" '..')).Path
Push-Location "$proj"
try {
  if (-not (Get-NetTCPConnection -LocalPort 6379 -State Listen -ErrorAction SilentlyContinue)) {
    Write-Warning 'Redis 未监听在 6379。请先启动 Redis（或修改 config.yml 指向可用 Redis）。'
  }
  $env:PORT = "$Port"
  $env:SERVER_ADDRESS = "http://localhost:$Port"
  go run .
} finally {
  Pop-Location
}
