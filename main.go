//go:build windows

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	appName    = "Brave App Forge"
	appVersion = "1.1.0"
)

const (
	WM_DESTROY   = 0x0002
	WM_COMMAND   = 0x0111
	WM_SETFONT   = 0x0030
	STM_SETIMAGE = 0x0172

	WS_OVERLAPPED  = 0x00000000
	WS_CAPTION     = 0x00C00000
	WS_SYSMENU     = 0x00080000
	WS_MINIMIZEBOX = 0x00020000
	WS_CHILD       = 0x40000000
	WS_VISIBLE     = 0x10000000
	WS_BORDER      = 0x00800000
	WS_TABSTOP     = 0x00010000

	ES_AUTOHSCROLL = 0x0080
	BS_PUSHBUTTON  = 0x00000000
	BS_GROUPBOX    = 0x00000007
	SS_LEFT        = 0x00000000
	SS_CENTER      = 0x00000001
	SS_BITMAP      = 0x0000000E
	SS_NOTIFY      = 0x00000100
	SS_CENTERIMAGE = 0x00000200

	SW_SHOW = 5

	COLOR_BTNFACE = 15
	IDC_ARROW     = 32512

	MB_OK          = 0x00000000
	MB_ICONERROR   = 0x00000010
	MB_ICONWARNING = 0x00000030
	MB_ICONINFO    = 0x00000040
	MB_YESNO       = 0x00000004
	IDYES          = 6

	IMAGE_BITMAP = 0

	OFN_PATHMUSTEXIST = 0x00000800
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_EXPLORER      = 0x00080000

	BIF_RETURNONLYFSDIRS = 0x0001
	BIF_NEWDIALOGSTYLE   = 0x0040

	CLSCTX_INPROC_SERVER     = 0x1
	COINIT_APARTMENTTHREADED = 0x2

	DEFAULT_GUI_FONT = 17
)

const (
	idURL          = 1001
	idFindIcons    = 1002
	idAppName      = 1003
	idBrave        = 1004
	idBrowseBrave  = 1005
	idIcon1        = 1010
	idIcon2        = 1011
	idIcon3        = 1012
	idIcon4        = 1013
	idCustomURL    = 1020
	idLoadURL      = 1021
	idBrowseImage  = 1022
	idSelected     = 1023
	idFolder       = 1030
	idBrowseFolder = 1031
	idOpenIcons    = 1032
	idCreate       = 1040
	idStatus       = 1041
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procShowWindow         = user32.NewProc("ShowWindow")
	procUpdateWindow       = user32.NewProc("UpdateWindow")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procLoadCursorW        = user32.NewProc("LoadCursorW")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen   = user32.NewProc("GetWindowTextLengthW")
	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procCreateDIBSection = gdi32.NewProc("CreateDIBSection")

	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procShellExecuteW        = shell32.NewProc("ShellExecuteW")
	procSHGetFolderPathW     = shell32.NewProc("SHGetFolderPathW")

	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree    = ole32.NewProc("CoTaskMemFree")
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct{ X, Y int32 }

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type OPENFILENAMEW struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

type BROWSEINFOW struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type RGBQUAD struct{ B, G, R, A byte }
type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]RGBQUAD
}

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	CLSID_ShellLink  = GUID{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	IID_IShellLinkW  = GUID{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	IID_IPersistFile = GUID{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

type IShellLinkW struct{ Vtbl *IShellLinkWVtbl }
type IShellLinkWVtbl struct {
	QueryInterface, AddRef, Release          uintptr
	GetPath, GetIDList, SetIDList            uintptr
	GetDescription, SetDescription           uintptr
	GetWorkingDirectory, SetWorkingDirectory uintptr
	GetArguments, SetArguments               uintptr
	GetHotkey, SetHotkey                     uintptr
	GetShowCmd, SetShowCmd                   uintptr
	GetIconLocation, SetIconLocation         uintptr
	SetRelativePath, Resolve, SetPath        uintptr
}

type IPersistFile struct{ Vtbl *IPersistFileVtbl }
type IPersistFileVtbl struct {
	QueryInterface, AddRef, Release                            uintptr
	GetClassID, IsDirty, Load, Save, SaveCompleted, GetCurFile uintptr
}

type candidate struct {
	Label  string
	URL    string
	Bytes  []byte
	Img    image.Image
	RawICO bool
}

type selectedIcon struct {
	Bytes       []byte
	Img         image.Image
	RawICO      bool
	Description string
}

var (
	hMain                                                           uintptr
	hURL, hAppName, hBrave, hCustomURL, hSelected, hFolder, hStatus uintptr
	hPreview                                                        [4]uintptr
	hPreviewLabel                                                   [4]uintptr
	previewBitmaps                                                  [4]uintptr
	candidates                                                      [4]*candidate
	selected                                                        *selectedIcon
	hDefaultFont                                                    uintptr
	hTitleFont                                                      uintptr
)

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func utf16WithNULs(s string) []uint16 {
	out := utf16.Encode([]rune(s))
	out = append(out, 0)
	return out
}

func loword(v uintptr) uint16 { return uint16(v & 0xFFFF) }

func messageBox(title, text string, flags uintptr) int {
	r, _, _ := procMessageBoxW.Call(hMain, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
	return int(r)
}

func setText(hwnd uintptr, s string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(u16(s))))
}

func getText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLen.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setStatus(s string) {
	setText(hStatus, s)
	procUpdateWindow.Call(hMain)
}

func createControl(class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(u16(class))),
		uintptr(unsafe.Pointer(u16(text))),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		hMain,
		uintptr(id),
		0, 0,
	)
	if hwnd != 0 && hDefaultFont != 0 {
		procSendMessageW.Call(hwnd, WM_SETFONT, hDefaultFont, 1)
	}
	return hwnd
}

func createUI() {
	hTitle := createControl("STATIC", appName, WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 14, 500, 42, 0)
	if hTitleFont != 0 {
		procSendMessageW.Call(hTitle, WM_SETFONT, hTitleFont, 1)
	}
	createControl("STATIC", "Create standalone Brave web-app shortcuts without the usual hassle.", WS_CHILD|WS_VISIBLE|SS_LEFT, 28, 58, 760, 22, 0)
	createControl("STATIC", "Brave Browser only — this tool is made specifically for Brave and does not create apps for Chrome, Edge, Firefox, or other browsers.", WS_CHILD|WS_VISIBLE|WS_BORDER|SS_LEFT, 28, 88, 790, 46, 0)

	createControl("STATIC", "Website URL", WS_CHILD|WS_VISIBLE|SS_LEFT, 28, 151, 160, 20, 0)
	hURL = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 173, 620, 26, idURL)
	createControl("BUTTON", "Find icon examples", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 660, 171, 158, 30, idFindIcons)

	createControl("STATIC", "App name", WS_CHILD|WS_VISIBLE|SS_LEFT, 28, 211, 160, 20, 0)
	hAppName = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 233, 790, 26, idAppName)

	createControl("STATIC", "Brave executable", WS_CHILD|WS_VISIBLE|SS_LEFT, 28, 271, 180, 20, 0)
	hBrave = createControl("EDIT", findBrave(), WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 293, 694, 26, idBrave)
	createControl("BUTTON", "Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 732, 291, 86, 30, idBrowseBrave)

	createControl("BUTTON", "Icon examples — click one to select it", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 28, 336, 790, 178, 0)
	xs := []int32{48, 242, 436, 630}
	for i := 0; i < 4; i++ {
		hPreview[i] = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER|SS_BITMAP|SS_NOTIFY|SS_CENTERIMAGE, xs[i], 366, 128, 104, idIcon1+i)
		hPreviewLabel[i] = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_CENTER, xs[i]-10, 476, 148, 24, 0)
	}
	setText(hPreviewLabel[0], "Enter a website above")

	createControl("BUTTON", "Custom icon", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 28, 528, 790, 104, 0)
	hCustomURL = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 42, 555, 526, 26, idCustomURL)
	createControl("BUTTON", "Load URL", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 578, 553, 94, 30, idLoadURL)
	createControl("BUTTON", "Browse image…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 682, 553, 122, 30, idBrowseImage)
	hSelected = createControl("STATIC", "Selected icon: none", WS_CHILD|WS_VISIBLE|SS_LEFT, 42, 594, 742, 24, idSelected)

	createControl("STATIC", "Shortcut folder", WS_CHILD|WS_VISIBLE|SS_LEFT, 28, 647, 180, 20, 0)
	hFolder = createControl("EDIT", desktopPath(), WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 669, 532, 26, idFolder)
	createControl("BUTTON", "Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 570, 667, 88, 30, idBrowseFolder)
	createControl("BUTTON", "Open Icons Folder", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 668, 667, 150, 30, idOpenIcons)

	createControl("BUTTON", "Create Brave App", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 28, 716, 190, 36, idCreate)
	hStatus = createControl("STATIC", "Ready.", WS_CHILD|WS_VISIBLE|SS_LEFT, 232, 722, 586, 26, idStatus)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		id := int(loword(wParam))
		switch id {
		case idFindIcons:
			onFindIcons()
		case idBrowseBrave:
			if p, ok := openFileDialog("Locate brave.exe", "Brave Browser (brave.exe)\x00brave.exe\x00Executable files (*.exe)\x00*.exe\x00All files\x00*.*\x00\x00"); ok {
				setText(hBrave, p)
			}
		case idLoadURL:
			onLoadCustomURL()
		case idBrowseImage:
			onBrowseImage()
		case idBrowseFolder:
			if p, ok := browseFolder("Choose where the Brave app shortcut should be created."); ok {
				setText(hFolder, p)
			}
		case idOpenIcons:
			dir, err := iconDir()
			if err == nil {
				_ = os.MkdirAll(dir, 0755)
				shellOpen(dir)
			}
		case idCreate:
			onCreate()
		case idIcon1, idIcon2, idIcon3, idIcon4:
			idx := id - idIcon1
			if idx >= 0 && idx < 4 && candidates[idx] != nil {
				c := candidates[idx]
				selected = &selectedIcon{Bytes: append([]byte(nil), c.Bytes...), Img: c.Img, RawICO: c.RawICO, Description: c.Label}
				setText(hSelected, "Selected icon: "+c.Label)
				setStatus("Icon selected. Ready to create the shortcut.")
			}
		}
		return 0
	case WM_DESTROY:
		cleanupBitmaps()
		if hTitleFont != 0 {
			procDeleteObject.Call(hTitleFont)
		}
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func normalizeURL(input string) (string, error) {
	v := strings.TrimSpace(input)
	aliases := map[string]string{
		"twitch":    "https://www.twitch.tv/",
		"youtube":   "https://www.youtube.com/",
		"github":    "https://github.com/",
		"discord":   "https://discord.com/app",
		"spotify":   "https://open.spotify.com/",
		"reddit":    "https://www.reddit.com/",
		"twitter":   "https://x.com/",
		"x":         "https://x.com/",
		"facebook":  "https://www.facebook.com/",
		"instagram": "https://www.instagram.com/",
	}
	if a, ok := aliases[strings.ToLower(v)]; ok {
		v = a
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("invalid website URL")
	}
	return u.String(), nil
}

func defaultName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "Web App"
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	first := strings.Split(host, ".")[0]
	known := map[string]string{"twitch": "Twitch", "youtube": "YouTube", "github": "GitHub", "discord": "Discord", "spotify": "Spotify", "reddit": "Reddit", "x": "X", "facebook": "Facebook", "instagram": "Instagram"}
	if n, ok := known[first]; ok {
		return n
	}
	if first == "" {
		return "Web App"
	}
	return strings.ToUpper(first[:1]) + first[1:]
}

func findBrave() string {
	pf := os.Getenv("ProgramFiles")
	pfx86 := os.Getenv("ProgramFiles(x86)")
	local := os.Getenv("LOCALAPPDATA")
	list := []string{
		filepath.Join(pf, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		filepath.Join(pfx86, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		filepath.Join(local, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
	}
	for _, p := range list {
		if p != "" {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

func desktopPath() string {
	// CSIDL_DESKTOPDIRECTORY handles redirected/OneDrive desktops better than simply using %USERPROFILE%\Desktop.
	buf := make([]uint16, 260)
	hr, _, _ := procSHGetFolderPathW.Call(0, 0x0010, 0, 0, uintptr(unsafe.Pointer(&buf[0])))
	if !failedHRESULT(hr) {
		if p := syscall.UTF16ToString(buf); p != "" {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Desktop")
	}
	return ""
}

func iconDir() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", errors.New("LOCALAPPDATA is not available")
	}
	return filepath.Join(local, "BraveAppForge", "Icons"), nil
}

func httpClient() *http.Client { return &http.Client{Timeout: 8 * time.Second} }

func fetchBytes(raw string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) BraveAppForge/"+appVersion)
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header.Get("Content-Type"), fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	return data, resp.Header.Get("Content-Type"), err
}

func fetchText(raw string) (string, error) {
	b, _, err := fetchBytes(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var (
	reLink    = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	reRel     = regexp.MustCompile(`(?is)\brel\s*=\s*["']([^"']+)["']`)
	reHref    = regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"']+)["']`)
	reMeta    = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	reProp    = regexp.MustCompile(`(?is)\b(?:property|name)\s*=\s*["']([^"']+)["']`)
	reContent = regexp.MustCompile(`(?is)\bcontent\s*=\s*["']([^"']+)["']`)
)

func candidateURLs(page string) []struct{ Label, URL string } {
	base, err := url.Parse(page)
	if err != nil {
		return nil
	}
	out := make([]struct{ Label, URL string }, 0, 8)
	seen := map[string]bool{}
	add := func(label, ref string) {
		ref = html.UnescapeString(strings.TrimSpace(ref))
		u, err := base.Parse(ref)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return
		}
		s := u.String()
		if seen[s] {
			return
		}
		seen[s] = true
		out = append(out, struct{ Label, URL string }{label, s})
	}
	if text, err := fetchText(page); err == nil {
		for _, tag := range reLink.FindAllString(text, -1) {
			rm := reRel.FindStringSubmatch(tag)
			hm := reHref.FindStringSubmatch(tag)
			if len(rm) < 2 || len(hm) < 2 {
				continue
			}
			rel := strings.ToLower(rm[1])
			if strings.Contains(rel, "apple-touch-icon") {
				add("Apple touch icon", hm[1])
			} else if strings.Contains(rel, "icon") {
				add("Website icon", hm[1])
			}
			if len(out) >= 5 {
				break
			}
		}
		for _, tag := range reMeta.FindAllString(text, -1) {
			pm := reProp.FindStringSubmatch(tag)
			cm := reContent.FindStringSubmatch(tag)
			if len(pm) < 2 || len(cm) < 2 {
				continue
			}
			p := strings.ToLower(pm[1])
			if p == "og:image" || p == "twitter:image" {
				add("Social image", cm[1])
				break
			}
		}
	}
	// Reliable PNG fallback from Google's favicon endpoint.
	domain := url.QueryEscape(base.Hostname())
	g := "https://www.google.com/s2/favicons?domain=" + domain + "&sz=256"
	if !seen[g] {
		out = append(out, struct{ Label, URL string }{"Google favicon", g})
	}
	add("Root favicon", "/favicon.ico")
	return out
}

func isICO(b []byte) bool { return len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0 }

func decodeAnyImage(b []byte) (image.Image, bool, error) {
	if isICO(b) {
		img, err := decodeICOPreview(b)
		if err != nil {
			return nil, true, nil
		} // valid ICO may use BMP entries; still accept it
		return img, true, nil
	}
	if len(b) >= 2 && b[0] == 'B' && b[1] == 'M' {
		img, err := decodeBMP(b)
		return img, false, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, false, err
}

func decodeICOPreview(b []byte) (image.Image, error) {
	if len(b) < 6 || !isICO(b) {
		return nil, errors.New("not ico")
	}
	count := int(binary.LittleEndian.Uint16(b[4:6]))
	if count < 1 || len(b) < 6+16*count {
		return nil, errors.New("invalid ico")
	}
	bestSize := -1
	var best []byte
	for i := 0; i < count; i++ {
		e := b[6+i*16 : 6+(i+1)*16]
		w := int(e[0])
		if w == 0 {
			w = 256
		}
		h := int(e[1])
		if h == 0 {
			h = 256
		}
		size := int(binary.LittleEndian.Uint32(e[8:12]))
		off := int(binary.LittleEndian.Uint32(e[12:16]))
		if size <= 0 || off < 0 || off+size > len(b) {
			continue
		}
		data := b[off : off+size]
		if len(data) >= 8 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) && w*h > bestSize {
			bestSize = w * h
			best = data
		}
	}
	if best == nil {
		return nil, errors.New("no png entry")
	}
	return png.Decode(bytes.NewReader(best))
}

func decodeBMP(b []byte) (image.Image, error) {
	if len(b) < 54 || b[0] != 'B' || b[1] != 'M' {
		return nil, errors.New("invalid bmp")
	}
	off := int(binary.LittleEndian.Uint32(b[10:14]))
	dib := binary.LittleEndian.Uint32(b[14:18])
	if dib < 40 || len(b) < 14+int(dib) {
		return nil, errors.New("unsupported bmp header")
	}
	w := int(int32(binary.LittleEndian.Uint32(b[18:22])))
	hs := int32(binary.LittleEndian.Uint32(b[22:26]))
	topDown := hs < 0
	h := int(hs)
	if h < 0 {
		h = -h
	}
	bpp := int(binary.LittleEndian.Uint16(b[28:30]))
	comp := binary.LittleEndian.Uint32(b[30:34])
	if w <= 0 || h <= 0 || (bpp != 24 && bpp != 32) || comp != 0 {
		return nil, errors.New("unsupported BMP (use 24/32-bit uncompressed BMP)")
	}
	row := ((w*bpp + 31) / 32) * 4
	if off < 0 || off+row*h > len(b) {
		return nil, errors.New("truncated bmp")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y
		if !topDown {
			sy = h - 1 - y
		}
		base := off + sy*row
		for x := 0; x < w; x++ {
			p := base + x*(bpp/8)
			bb, gg, rr := b[p], b[p+1], b[p+2]
			aa := byte(255)
			if bpp == 32 && b[p+3] != 0 {
				aa = b[p+3]
			}
			img.SetNRGBA(x, y, color.NRGBA{R: rr, G: gg, B: bb, A: aa})
		}
	}
	return img, nil
}

func onFindIcons() {
	raw, err := normalizeURL(getText(hURL))
	if err != nil {
		messageBox(appName, "Enter a valid website URL, for example https://www.twitch.tv/", MB_OK|MB_ICONWARNING)
		return
	}
	setText(hURL, raw)
	if strings.TrimSpace(getText(hAppName)) == "" {
		setText(hAppName, defaultName(raw))
	}
	clearCandidates()
	setStatus("Looking for icon examples…")
	urls := candidateURLs(raw)
	slot := 0
	for _, item := range urls {
		if slot >= 4 {
			break
		}
		setStatus("Trying " + item.Label + "…")
		b, _, err := fetchBytes(item.URL)
		if err != nil {
			continue
		}
		img, rawICO, err := decodeAnyImage(b)
		if err != nil || img == nil {
			continue
		}
		c := &candidate{Label: item.Label, URL: item.URL, Bytes: b, Img: img, RawICO: rawICO}
		candidates[slot] = c
		setPreview(slot, img, item.Label)
		slot++
	}
	if slot == 0 {
		setText(hPreviewLabel[0], "No examples found")
		setStatus("No icon examples found. You can use a custom image URL or browse for an image.")
		return
	}
	c := candidates[0]
	selected = &selectedIcon{Bytes: append([]byte(nil), c.Bytes...), Img: c.Img, RawICO: c.RawICO, Description: c.Label}
	setText(hSelected, "Selected icon: "+c.Label)
	setStatus("Found " + strconv.Itoa(slot) + " icon example(s). The first one is selected.")
}

func onLoadCustomURL() {
	raw := strings.TrimSpace(getText(hCustomURL))
	if raw == "" {
		return
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	setStatus("Downloading custom icon…")
	b, _, err := fetchBytes(raw)
	if err != nil {
		messageBox(appName, "Could not download that image.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		setStatus("Could not load custom icon.")
		return
	}
	img, ico, err := decodeAnyImage(b)
	if err != nil || (img == nil && !ico) {
		messageBox(appName, "That URL did not return a supported image.\n\nSupported: PNG, JPG/JPEG, GIF, BMP, and ICO.", MB_OK|MB_ICONWARNING)
		return
	}
	selected = &selectedIcon{Bytes: b, Img: img, RawICO: ico, Description: "custom URL"}
	setText(hSelected, "Selected icon: custom URL")
	setStatus("Custom icon loaded.")
}

func onBrowseImage() {
	p, ok := openFileDialog("Choose an image", "Image files\x00*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.ico\x00PNG files\x00*.png\x00JPEG files\x00*.jpg;*.jpeg\x00ICO files\x00*.ico\x00All files\x00*.*\x00\x00")
	if !ok {
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		messageBox(appName, "Could not read that image file.", MB_OK|MB_ICONERROR)
		return
	}
	img, ico, err := decodeAnyImage(b)
	if err != nil || (img == nil && !ico) {
		messageBox(appName, "That file is not a supported image.\n\nSupported: PNG, JPG/JPEG, GIF, BMP, and ICO.", MB_OK|MB_ICONWARNING)
		return
	}
	selected = &selectedIcon{Bytes: b, Img: img, RawICO: ico, Description: filepath.Base(p)}
	setText(hSelected, "Selected icon: "+filepath.Base(p))
	setStatus("Custom icon selected.")
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	bad := `<>:"/\\|?*`
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(bad, r) {
			return '_'
		}
		return r
	}, s)
	s = strings.TrimRight(s, ". ")
	if s == "" {
		s = "Brave App"
	}
	return s
}

func onCreate() {
	raw, err := normalizeURL(getText(hURL))
	if err != nil {
		messageBox(appName, "Enter a valid website URL.", MB_OK|MB_ICONWARNING)
		return
	}
	setText(hURL, raw)
	name := strings.TrimSpace(getText(hAppName))
	if name == "" {
		name = defaultName(raw)
		setText(hAppName, name)
	}
	brave := strings.TrimSpace(getText(hBrave))
	st, err := os.Stat(brave)
	if err != nil || st.IsDir() || strings.ToLower(filepath.Base(brave)) != "brave.exe" {
		messageBox(appName, "Brave could not be found. Browse to your regular Brave installation and select brave.exe.\n\nThis tool only creates apps for Brave Browser.", MB_OK|MB_ICONWARNING)
		return
	}
	folder := strings.TrimSpace(getText(hFolder))
	st, err = os.Stat(folder)
	if err != nil || !st.IsDir() {
		messageBox(appName, "The shortcut folder does not exist.", MB_OK|MB_ICONWARNING)
		return
	}

	if selected == nil {
		if messageBox(appName, "No icon is selected. The shortcut will use the Brave icon instead. Continue?", MB_YESNO|MB_ICONWARNING) != IDYES {
			return
		}
	}

	setStatus("Creating Brave app shortcut…")
	sn := safeName(name)
	shortcut := filepath.Join(folder, sn+".lnk")
	iconPath := ""
	if selected != nil {
		dir, err := iconDir()
		if err != nil {
			messageBox(appName, err.Error(), MB_OK|MB_ICONERROR)
			return
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			messageBox(appName, "Could not create the icons folder.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
			return
		}
		sum := sha256.Sum256(selected.Bytes)
		iconPath = filepath.Join(dir, fmt.Sprintf("%s_%x.ico", sn, sum[:4]))
		if selected.RawICO {
			if err := os.WriteFile(iconPath, selected.Bytes, 0644); err != nil {
				messageBox(appName, "Could not save the icon.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
				return
			}
		} else {
			if selected.Img == nil {
				messageBox(appName, "The selected image could not be converted.", MB_OK|MB_ICONERROR)
				return
			}
			if err := writeICO(iconPath, selected.Img); err != nil {
				messageBox(appName, "Could not convert the image to ICO.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
				return
			}
		}
	}
	args := `--app="` + raw + `"`
	desc := "Brave web app: " + raw
	if err := createShortcut(shortcut, brave, args, filepath.Dir(brave), iconPath, desc); err != nil {
		messageBox(appName, "Could not create the Brave app shortcut.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		setStatus("Something went wrong.")
		return
	}
	setStatus("Done.")
	msg := "Created:\n" + shortcut
	if iconPath != "" {
		dir, _ := iconDir()
		msg += "\n\nThe icon is stored in:\n" + dir
	}
	messageBox(appName, msg, MB_OK|MB_ICONINFO)
}

func resizeFit(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}
	scale := float64(size) / float64(sw)
	if float64(sh)*scale > float64(size) {
		scale = float64(size) / float64(sh)
	}
	dw := int(float64(sw)*scale + 0.5)
	dh := int(float64(sh)*scale + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	ox, oy := (size-dw)/2, (size-dh)/2
	for y := 0; y < dh; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(dh) - 0.5
		y0 := int(fy)
		if y0 < 0 {
			y0 = 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		wy := fy - float64(y0)
		if wy < 0 {
			wy = 0
		}
		for x := 0; x < dw; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(dw) - 0.5
			x0 := int(fx)
			if x0 < 0 {
				x0 = 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			wx := fx - float64(x0)
			if wx < 0 {
				wx = 0
			}
			c00 := color.NRGBAModel.Convert(src.At(sb.Min.X+x0, sb.Min.Y+y0)).(color.NRGBA)
			c10 := color.NRGBAModel.Convert(src.At(sb.Min.X+x1, sb.Min.Y+y0)).(color.NRGBA)
			c01 := color.NRGBAModel.Convert(src.At(sb.Min.X+x0, sb.Min.Y+y1)).(color.NRGBA)
			c11 := color.NRGBAModel.Convert(src.At(sb.Min.X+x1, sb.Min.Y+y1)).(color.NRGBA)
			mix := func(a, b, c, d byte) byte {
				top := float64(a)*(1-wx) + float64(b)*wx
				bot := float64(c)*(1-wx) + float64(d)*wx
				v := top*(1-wy) + bot*wy
				if v < 0 {
					v = 0
				}
				if v > 255 {
					v = 255
				}
				return byte(v + 0.5)
			}
			dst.SetNRGBA(ox+x, oy+y, color.NRGBA{mix(c00.R, c10.R, c01.R, c11.R), mix(c00.G, c10.G, c01.G, c11.G), mix(c00.B, c10.B, c01.B, c11.B), mix(c00.A, c10.A, c01.A, c11.A)})
		}
	}
	return dst
}

func writeICO(path string, src image.Image) error {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	pngs := make([][]byte, len(sizes))
	for i, s := range sizes {
		img := resizeFit(src, s)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		pngs[i] = buf.Bytes()
	}
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(sizes)))
	offset := uint32(6 + 16*len(sizes))
	for i, s := range sizes {
		wb, hb := byte(s), byte(s)
		if s == 256 {
			wb = 0
			hb = 0
		}
		out.WriteByte(wb)
		out.WriteByte(hb)
		out.WriteByte(0)
		out.WriteByte(0)
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))
		_ = binary.Write(&out, binary.LittleEndian, uint16(32))
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(pngs[i])))
		_ = binary.Write(&out, binary.LittleEndian, offset)
		offset += uint32(len(pngs[i]))
	}
	for _, p := range pngs {
		out.Write(p)
	}
	return os.WriteFile(path, out.Bytes(), 0644)
}

func setPreview(i int, img image.Image, label string) {
	if i < 0 || i >= 4 {
		return
	}
	if previewBitmaps[i] != 0 {
		procDeleteObject.Call(previewBitmaps[i])
		previewBitmaps[i] = 0
	}
	hb, err := hBitmapFromImage(img, 120, 96)
	if err == nil {
		previewBitmaps[i] = hb
		procSendMessageW.Call(hPreview[i], STM_SETIMAGE, IMAGE_BITMAP, hb)
	}
	setText(hPreviewLabel[i], label)
}

func clearCandidates() {
	for i := 0; i < 4; i++ {
		candidates[i] = nil
		setText(hPreviewLabel[i], "")
		if previewBitmaps[i] != 0 {
			procSendMessageW.Call(hPreview[i], STM_SETIMAGE, IMAGE_BITMAP, 0)
			procDeleteObject.Call(previewBitmaps[i])
			previewBitmaps[i] = 0
		}
	}
}

func cleanupBitmaps() {
	for i := 0; i < 4; i++ {
		if previewBitmaps[i] != 0 {
			procDeleteObject.Call(previewBitmaps[i])
			previewBitmaps[i] = 0
		}
	}
}

func hBitmapFromImage(src image.Image, width, height int) (uintptr, error) {
	// Fit image into a white preview canvas, then create a 32-bit DIB section.
	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(canvas.Pix); i += 4 {
		canvas.Pix[i] = 255
		canvas.Pix[i+1] = 255
		canvas.Pix[i+2] = 255
		canvas.Pix[i+3] = 255
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return 0, errors.New("empty image")
	}
	scale := float64(width-8) / float64(sw)
	if float64(sh)*scale > float64(height-8) {
		scale = float64(height-8) / float64(sh)
	}
	dw := int(float64(sw)*scale + 0.5)
	dh := int(float64(sh)*scale + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	ox, oy := (width-dw)/2, (height-dh)/2
	resized := resizeFit(src, max(dw, dh))
	// resizeFit is square; draw the central content scaled again via nearest sampling to exact dw/dh.
	rb := resized.Bounds()
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := rb.Min.X + x*rb.Dx()/dw
			sy := rb.Min.Y + y*rb.Dy()/dh
			c := color.NRGBAModel.Convert(resized.At(sx, sy)).(color.NRGBA)
			a := float64(c.A) / 255.0
			bg := color.NRGBA{255, 255, 255, 255}
			canvas.SetNRGBA(ox+x, oy+y, color.NRGBA{byte(float64(c.R)*a + float64(bg.R)*(1-a)), byte(float64(c.G)*a + float64(bg.G)*(1-a)), byte(float64(c.B)*a + float64(bg.B)*(1-a)), 255})
		}
	}
	bi := BITMAPINFO{}
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(BITMAPINFOHEADER{}))
	bi.BmiHeader.BiWidth = int32(width)
	bi.BmiHeader.BiHeight = -int32(height)
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = 0
	bi.BmiHeader.BiSizeImage = uint32(width * height * 4)
	var bits uintptr
	hb, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hb == 0 || bits == 0 {
		return 0, errors.New("CreateDIBSection failed")
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := canvas.NRGBAAt(x, y)
			idx := (y*width + x) * 4
			dst[idx] = p.B
			dst[idx+1] = p.G
			dst[idx+2] = p.R
			dst[idx+3] = 255
		}
	}
	return hb, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func openFileDialog(title, filter string) (string, bool) {
	fileBuf := make([]uint16, 32768)
	filterBuf := utf16WithNULs(filter)
	titlePtr := u16(title)
	ofn := OPENFILENAMEW{}
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.HwndOwner = hMain
	ofn.LpstrFilter = &filterBuf[0]
	ofn.NFilterIndex = 1
	ofn.LpstrFile = &fileBuf[0]
	ofn.NMaxFile = uint32(len(fileBuf))
	ofn.LpstrTitle = titlePtr
	ofn.Flags = OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST
	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return "", false
	}
	return syscall.UTF16ToString(fileBuf), true
}

func browseFolder(title string) (string, bool) {
	display := make([]uint16, 260)
	bi := BROWSEINFOW{HwndOwner: hMain, PszDisplayName: &display[0], LpszTitle: u16(title), UlFlags: BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", false
	}
	defer procCoTaskMemFree.Call(pidl)
	path := make([]uint16, 32768)
	r, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		return "", false
	}
	return syscall.UTF16ToString(path), true
}

func shellOpen(path string) {
	procShellExecuteW.Call(hMain, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(path))), 0, 0, SW_SHOW)
}

func failedHRESULT(hr uintptr) bool { return int32(uint32(hr)) < 0 }
func hrErr(where string, hr uintptr) error {
	return fmt.Errorf("%s failed (HRESULT 0x%08X)", where, uint32(hr))
}

func createShortcut(shortcut, target, args, workdir, icon, desc string) error {
	var sl *IShellLinkW
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&CLSID_ShellLink)), 0, CLSCTX_INPROC_SERVER, uintptr(unsafe.Pointer(&IID_IShellLinkW)), uintptr(unsafe.Pointer(&sl)))
	if failedHRESULT(hr) || sl == nil {
		return hrErr("CoCreateInstance", hr)
	}
	defer syscall.SyscallN(sl.Vtbl.Release, uintptr(unsafe.Pointer(sl)))

	callStr := func(fn uintptr, s string) error {
		p := u16(s)
		hr, _, _ := syscall.SyscallN(fn, uintptr(unsafe.Pointer(sl)), uintptr(unsafe.Pointer(p)))
		runtime.KeepAlive(p)
		if failedHRESULT(hr) {
			return hrErr("ShellLink", hr)
		}
		return nil
	}
	if err := callStr(sl.Vtbl.SetPath, target); err != nil {
		return err
	}
	if err := callStr(sl.Vtbl.SetArguments, args); err != nil {
		return err
	}
	if workdir != "" {
		if err := callStr(sl.Vtbl.SetWorkingDirectory, workdir); err != nil {
			return err
		}
	}
	if desc != "" {
		if err := callStr(sl.Vtbl.SetDescription, desc); err != nil {
			return err
		}
	}
	if icon != "" {
		p := u16(icon)
		hr, _, _ := syscall.SyscallN(sl.Vtbl.SetIconLocation, uintptr(unsafe.Pointer(sl)), uintptr(unsafe.Pointer(p)), 0)
		runtime.KeepAlive(p)
		if failedHRESULT(hr) {
			return hrErr("SetIconLocation", hr)
		}
	}
	var pf *IPersistFile
	hr, _, _ = syscall.SyscallN(sl.Vtbl.QueryInterface, uintptr(unsafe.Pointer(sl)), uintptr(unsafe.Pointer(&IID_IPersistFile)), uintptr(unsafe.Pointer(&pf)))
	if failedHRESULT(hr) || pf == nil {
		return hrErr("QueryInterface(IPersistFile)", hr)
	}
	defer syscall.SyscallN(pf.Vtbl.Release, uintptr(unsafe.Pointer(pf)))
	p := u16(shortcut)
	hr, _, _ = syscall.SyscallN(pf.Vtbl.Save, uintptr(unsafe.Pointer(pf)), uintptr(unsafe.Pointer(p)), 1)
	runtime.KeepAlive(p)
	if failedHRESULT(hr) {
		return hrErr("IPersistFile.Save", hr)
	}
	return nil
}

func writeCrashLog(v any) string {
	dir, err := iconDir()
	if err != nil {
		return ""
	}
	root := filepath.Dir(dir)
	_ = os.MkdirAll(root, 0755)
	p := filepath.Join(root, "BraveAppForge.log")
	_ = os.WriteFile(p, []byte(fmt.Sprintf("Brave App Forge %s\r\nStartup/runtime error: %v\r\n", appVersion, v)), 0644)
	return p
}

func main() {
	runtime.LockOSThread()
	defer func() {
		if v := recover(); v != nil {
			p := writeCrashLog(v)
			t := fmt.Sprintf("The app hit an unexpected error:\n\n%v", v)
			if p != "" {
				t += "\n\nA log was written to:\n" + p
			}
			messageBox(appName, t, MB_OK|MB_ICONERROR)
		}
	}()

	procSetProcessDPIAware.Call()
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	if !failedHRESULT(hr) {
		defer procCoUninitialize.Call()
	}

	hInst, _, _ := procGetModuleHandleW.Call(0)
	cur, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	className := u16("BraveAppForgeWindowClass")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: syscall.NewCallback(wndProc), HInstance: hInst, HCursor: cur, HbrBackground: uintptr(COLOR_BTNFACE + 1), LpszClassName: className}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		messageBox(appName, "Could not register the application window.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		return
	}

	hDefaultFont, _, _ = procGetStockObject.Call(DEFAULT_GUI_FONT)
	// Create a Segoe UI title font. Failure simply falls back to the default font.
	fontHeight := int32(-28)
	hTitleFont, _, _ = procCreateFontW.Call(uintptr(fontHeight), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16("Segoe UI"))))

	width, height := 860, 810
	sw, _, _ := procGetSystemMetrics.Call(0)
	sh, _, _ := procGetSystemMetrics.Call(1)
	x := int32((int(sw) - width) / 2)
	y := int32((int(sh) - height) / 2)
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	style := uint32(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX)
	hMain, _, err = procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(u16(appName+" "+appVersion))), uintptr(style), uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, hInst, 0)
	if hMain == 0 {
		messageBox(appName, "Could not create the application window.\n\n"+err.Error(), MB_OK|MB_ICONERROR)
		return
	}
	createUI()
	procShowWindow.Call(hMain, SW_SHOW)
	procUpdateWindow.Call(hMain)

	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func init() {
	// Register standard decoders explicitly so image.Decode knows them.
	_ = gif.GIF{}
	_ = jpeg.DefaultQuality
}
