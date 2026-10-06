// internal/admin/admin.go
package admin

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsAdmin 判断当前进程是否以管理员运行
func IsAdmin() bool {
	// 方法一：尝试写 hosts 文件
	f, err := os.OpenFile(`C:\Windows\System32\drivers\etc\hosts`, os.O_WRONLY, 0)
	if err == nil {
		f.Close()
		return true
	}
	return false
}

// IsAdmin2 判断当前进程是否以管理员运行（Win32 API）
func IsAdmin2() bool {
	var sid *windows.SID
	// BUILTIN\Administrators 的 SID: S-1-5-32-544
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// 获取当前进程的 Token
	token := windows.Token(0) // 0 是伪句柄，代表当前进程
	// 注意：部分版本可能需要 windows.GetCurrentProcess() 后再 OpenProcessToken
	// 但多数情况下直接用 Token(0) 即可

	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// RelaunchAsAdmin 以管理员身份重新启动当前程序
func RelaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	verb := "runas"
	args := ""
	if len(os.Args) > 1 {
		args = joinArgs(os.Args[1:])
	}

	// 用 ShellExecute 提权
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	filePtr, _ := syscall.UTF16PtrFromString(exe)
	argsPtr, _ := syscall.UTF16PtrFromString(args)
	cwd, _ := os.Getwd()
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)

	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		return syscall.EINVAL
	}
	return nil
}

func joinArgs(args []string) string {
	// 简单拼接，如果参数含空格用引号包起来
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		if contains(a, " ") {
			result += `"` + a + `"`
		} else {
			result += a
		}
	}
	return result
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
