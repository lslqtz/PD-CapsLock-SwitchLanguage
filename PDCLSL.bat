@echo off

if "%1"=="h" goto loop
mshta vbscript:CreateObject("Shell.Application").ShellExecute("cmd.exe","/c %~s0 h","","runas",0)(window.close)&&exit
exit

:loop
	"C:\Users\VM\E\PD-CapsLock-SwitchLanguage.exe"
	timeout /t 1 >nul
goto loop
