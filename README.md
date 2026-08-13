# Brave App Forge

Brave App Forge is a small **portable Windows utility for Brave Browser only**. It creates standalone Brave web-app shortcuts for sites that do not offer their own install button.

For example, enter `twitch` or `https://www.twitch.tv/`, choose an icon, and Brave App Forge creates a Windows shortcut that launches:

```text
brave.exe --app="https://www.twitch.tv/"
```

## Features

- Made specifically for **Brave Browser**
- Portable: no installer and no background service
- No Python, PowerShell, .NET, or other runtime is required to start the app
- Accepts a full website URL or shortcuts such as `twitch`, `youtube`, `github`, `discord`, and `spotify`
- Finds several icon examples from the website and a favicon fallback
- Load your own icon from an image URL
- Browse for a local PNG, JPG/JPEG, GIF, BMP, or ICO file
- Converts PNG/JPG/GIF/BMP images to a multi-size Windows `.ico`
- Stores generated icons in `%LOCALAPPDATA%\BraveAppForge\Icons`
- **Open Icons Folder** button makes it easy to remove generated icons later
- Automatically looks for normal Brave installations in Program Files and Local AppData
- Creates normal Windows `.lnk` shortcuts using the native Windows Shell API

## Requirements

- Windows 10 or Windows 11, 64-bit
- Brave Browser installed normally (not required to be installed in a specific folder; you can browse to `brave.exe`)
- Internet access only when using **Find icon examples** or **Load URL**

## Usage

1. Run `BraveAppForge.exe`.
2. Enter a website, such as `twitch` or `https://www.twitch.tv/`.
3. Click **Find icon examples** if you want the app to look for icons automatically.
4. Click an icon example, load an image URL, or click **Browse image…**.
5. Check that the detected Brave executable points to `brave.exe`.
6. Choose the folder where the shortcut should be created. The Desktop is used by default.
7. Click **Create Brave App**.

The generated shortcut starts the website in Brave's app mode without the normal tab bar and address bar.

## Generated icons

Converted icons are stored here:

```text
%LOCALAPPDATA%\BraveAppForge\Icons
```

Use **Open Icons Folder** in the application if you want to view or delete them.

Do not delete an icon while a shortcut still uses it, otherwise Windows may fall back to another icon for that shortcut.

## Portable

Brave App Forge does not install itself. You can keep `BraveAppForge.exe` anywhere you like, including a USB drive.

Generated shortcut icons are intentionally stored in your local AppData folder instead of next to the portable EXE. This means shortcuts keep their icons even if you move Brave App Forge itself.

## Building from source

The project uses only the Go standard library and native Windows APIs.

Install Go, then run:

```bat
build.bat
```

Or build manually:

```bat
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags="-H=windowsgui -s -w" -o BraveAppForge.exe .
```

## Notes

The executable is not digitally signed. Windows SmartScreen may therefore show an "Unknown publisher" warning on some systems. The source code is included so the binary can be rebuilt and inspected.

Brave App Forge is not affiliated with or endorsed by Brave Software, Twitch, YouTube, or any other website.

## License

MIT License. See `LICENSE`.
