package main

import (
	"log"
	"syscall"
	"unsafe"
	"time"

	"golang.org/x/sys/windows"
)

const (
	WH_KEYBOARD_LL            = 13
	WM_KEYDOWN                = 0x0100
	WM_SYSKEYDOWN             = 0x0104
	VK_CAPITAL                = 0x14
	WM_INPUTLANGCHANGEREQUEST = 0x0050
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	setWindowsHookExW      = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHookEx    = user32.NewProc("UnhookWindowsHookEx")
	callNextHookEx         = user32.NewProc("CallNextHookEx")
	getMessageW            = user32.NewProc("GetMessageW")
	translateMessage       = user32.NewProc("TranslateMessage")
	dispatchMessage        = user32.NewProc("DispatchMessageW")
	getForegroundWindow    = user32.NewProc("GetForegroundWindow")
	postMessageW           = user32.NewProc("PostMessageW")
	getKeyState            = user32.NewProc("GetKeyState")
)

var (
	hookID           windows.Handle
	lastKeyPressTime time.Time
	debounceDuration = 100 * time.Millisecond
)


type MSG struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X, Y int32
}

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

var keyboardHookProc = syscall.NewCallback(func(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == 0 {
		kbdStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		if (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN) && kbdStruct.VkCode == VK_CAPITAL {
			if !isCapsLockOn() {
				if time.Since(lastKeyPressTime) > debounceDuration {
					handleCapsLock()
					lastKeyPressTime = time.Now()
				}
				return 1
			}
		}
	}

	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
})

func isCapsLockOn() bool {
	ret, _, _ := getKeyState.Call(uintptr(VK_CAPITAL))
	return ret&1 != 0
}

func handleCapsLock() {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return
	}
	postMessageW.Call(hwnd, WM_INPUTLANGCHANGEREQUEST, 0, 1)
}

func main() {
	log.Println("程序启动...")

	hookIDPtr, _, _ := setWindowsHookExW.Call(
		WH_KEYBOARD_LL,
		keyboardHookProc,
		0,
		0,
	)
	hookID = windows.Handle(hookIDPtr)

	if hookID == 0 {
		winErr := windows.GetLastError()
		log.Fatalf("错误: 设置键盘钩子失败 (Error: %s)", winErr)
	}

	defer func() {
		log.Println("正在卸载键盘钩子...")
		ret, _, _ := unhookWindowsHookEx.Call(uintptr(hookID))
		if ret == 0 {
			log.Println("错误: 卸载键盘钩子失败")
		} else {
			log.Println("已卸载键盘钩子")
		}
	}()

	log.Println("已成功设置键盘钩子")

	var msg MSG
	for {
		ret, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int(ret) {
		case 0:
			log.Println("收到 WM_QUIT, 程序退出")
			return
		case -1:
			log.Println("错误: GetMessage 失败")
			return
		default:
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}