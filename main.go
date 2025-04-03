package main

import (
	"log"
	"syscall"
	"time"
	"unsafe"

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
	user32              = windows.NewLazySystemDLL("user32.dll")
	setWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	getMessageW         = user32.NewProc("GetMessageW")
	translateMessage    = user32.NewProc("TranslateMessage")
	dispatchMessage     = user32.NewProc("DispatchMessageW")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	postMessageW        = user32.NewProc("PostMessageW")
	getKeyState         = user32.NewProc("GetKeyState")
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
	// 捕获 panic，防止因异常导致整个进程崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered in keyboardHookProc: %v", r)
		}
	}()

	// 只有当 nCode 等于 0 时处理消息
	if nCode == 0 {
		// 对指针进行基本校验
		if lParam == 0 {
			ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
			return ret
		}
		kbdStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		if (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN) && kbdStruct.VkCode == VK_CAPITAL {
			if !isCapsLockOn() {
				if time.Since(lastKeyPressTime) > debounceDuration {
					handleCapsLock()
					lastKeyPressTime = time.Now()
				}
				// 消费掉该按键事件，不传递到下一个钩子
				return 1
			}
		}
	}
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
})

func isCapsLockOn() bool {
	ret, _, err := getKeyState.Call(uintptr(VK_CAPITAL))
	// 检查返回错误
	if err != nil && err != syscall.Errno(0) {
		log.Printf("getKeyState error: %v", err)
		return false // 默认返回关闭状态
	}
	return ret&1 != 0
}

func handleCapsLock() {
	hwnd, _, err := getForegroundWindow.Call()
	if hwnd == 0 {
		log.Printf("getForegroundWindow error: %v", err)
		return
	}
	ret, _, err := postMessageW.Call(hwnd, WM_INPUTLANGCHANGEREQUEST, 0, 1)
	if ret == 0 {
		log.Printf("postMessageW error: %v", err)
	}
}

func main() {
	log.Println("程序启动...")

	hookIDPtr, _, err := setWindowsHookExW.Call(
		WH_KEYBOARD_LL,
		keyboardHookProc,
		0,
		0,
	)
	if hookIDPtr == 0 {
		winErr := windows.GetLastError()
		log.Fatalf("错误: 设置键盘钩子失败 (Error: %v, %v)", winErr, err)
	}
	hookID = windows.Handle(hookIDPtr)

	defer func() {
		log.Println("正在卸载键盘钩子...")
		ret, _, err := unhookWindowsHookEx.Call(uintptr(hookID))
		if ret == 0 {
			log.Printf("错误: 卸载键盘钩子失败, 错误信息: %v", err)
		} else {
			log.Println("已卸载键盘钩子")
		}
	}()

	log.Println("已成功设置键盘钩子")

	var msg MSG
	for {
		ret, _, err := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			log.Println("收到 WM_QUIT, 程序退出")
			break
		} else if ret == uintptr(^uint(0)) { // -1
			log.Printf("错误: GetMessageW 调用失败, 错误信息: %v", err)
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
