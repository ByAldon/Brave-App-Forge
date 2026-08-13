@echo off
setlocal
cd /d "%~dp0"
where go >nul 2>nul
if errorlevel 1 (
  echo Go was not found. Install Go from https://go.dev/dl/ and try again.
  pause
  exit /b 1
)
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags="-H=windowsgui -s -w" -o BraveAppForge.exe .
if errorlevel 1 (
  echo Build failed.
  pause
  exit /b 1
)
echo Built BraveAppForge.exe
pause
