# Changelog

## 1.1.0

- Rebuilt the application as a native Windows GUI executable.
- Removed the PowerShell/WinForms launcher used by the first build.
- The application now starts without requiring Python, PowerShell, .NET, or another runtime.
- Windows shortcut creation now uses the native Windows Shell COM API.
- Kept Brave-only validation, website icon examples, custom image URLs, local image browsing, image-to-ICO conversion, and the Open Icons Folder button.
- Added a console diagnostic build under `diagnostics/` in the source ZIP for troubleshooting only.
