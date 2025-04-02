# PD-CapsLock-SwitchLanguage
在按下 CAPS LOCK 时, 切换输入法, 适用于关闭了键盘同步的 Parallels Desktop VM.  
计划任务附在 xml 内, 但须替换自己的 UserID.

```
GOOS=windows GOARCH=arm64 go build -ldflags="-H windowsgui"
```
