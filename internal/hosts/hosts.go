package hosts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const HostsPath = `C:\Windows\System32\drivers\etc\hosts`

var TargetHosts = []string{
	"translate.googleapis.com",
	"translate.googleapis.cn",
	"translate.google.com",
	"translate.google.cn",
}

var translateLineRe = regexp.MustCompile(`(?i)^\s*[^\s#]+\s+translate\.google\S*.*$`)

// Update 备份 hosts、删除旧行、追加新行、刷新 DNS。
// 写入前去掉只读/系统/隐藏属性，写入后还原。
func Update(ip string) error {
	// 1. 备份
	if data, err := os.ReadFile(HostsPath); err == nil {
		backup := fmt.Sprintf("%s.bak_%s", HostsPath, time.Now().Format("20060102_150405"))
		if err := os.WriteFile(backup, data, 0644); err == nil {
			fmt.Printf("已备份 hosts 到: %s\n", backup)
		}
	}

	// 2. 读现有内容
	data, err := os.ReadFile(HostsPath)
	if err != nil {
		return fmt.Errorf("读取 hosts 失败: %w", err)
	}

	// 3. 删除已有 translate.google 行
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if translateLineRe.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}

	// 4. 构造新内容
	var sb strings.Builder
	sb.WriteString(strings.Join(kept, "\n"))
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("# Google Translate Fix - %s\r\n", time.Now().Format("2006-01-02 15:04:05")))
	for _, h := range TargetHosts {
		sb.WriteString(fmt.Sprintf("%s\t%s\r\n", ip, h))
	}

	// 5. 去掉只读/系统/隐藏属性
	origAttrs := getAttributes()
	clearReadOnlyAttributes()

	// 6. 写回
	writeErr := os.WriteFile(HostsPath, []byte(sb.String()), 0644)

	// 7. 还原属性（无论写入成功与否）
	restoreAttributes(origAttrs)

	if writeErr != nil {
		return fmt.Errorf("写入 hosts 失败: %w", writeErr)
	}
	fmt.Printf("已更新 hosts，指向 %s\n", ip)

	// 8. 刷新 DNS
	cmd := exec.Command("ipconfig", "/flushdns")
	_ = cmd.Run()
	fmt.Println("已刷新 DNS 缓存。")

	return nil
}

// getAttributes 返回 hosts 文件当前的属性（只读/系统/隐藏）。
func getAttributes() fileAttrs {
	var a fileAttrs
	out, err := exec.Command("attrib", HostsPath).Output()
	if err != nil {
		return a
	}
	s := string(out)
	a.readOnly = strings.Contains(s, "R")
	a.system = strings.Contains(s, "S")
	a.hidden = strings.Contains(s, "H")
	return a
}

// clearReadOnlyAttributes 去掉只读/系统/隐藏属性。
func clearReadOnlyAttributes() {
	_ = exec.Command("attrib", "-R", "-S", "-H", HostsPath).Run()
}

// restoreAttributes 还原属性。
func restoreAttributes(a fileAttrs) {
	var flags []string
	if a.readOnly {
		flags = append(flags, "+R")
	}
	if a.system {
		flags = append(flags, "+S")
	}
	if a.hidden {
		flags = append(flags, "+H")
	}
	if len(flags) == 0 {
		return
	}
	args := append(flags, HostsPath)
	_ = exec.Command("attrib", args...).Run()
}

type fileAttrs struct {
	readOnly bool
	system   bool
	hidden   bool
}

// Restore 从最新的备份恢复 hosts。
func Restore() error {
	// 1. 找到所有 hosts.bak_* 文件
	dir := filepath.Dir(HostsPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	var backups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "hosts.bak_") {
			backups = append(backups, filepath.Join(dir, name))
		}
	}
	if len(backups) == 0 {
		return fmt.Errorf("没有找到任何备份文件")
	}

	// 2. 按文件名排序（时间戳在文件名里，字典序 == 时间序）
	sort.Strings(backups)
	latest := backups[len(backups)-1]
	fmt.Printf("使用最新备份: %s\n", latest)

	// 3. 读备份内容
	data, err := os.ReadFile(latest)
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}

	// 4. 去掉只读属性，写入 hosts，再还原
	origAttrs := getAttributes()
	clearReadOnlyAttributes()

	writeErr := os.WriteFile(HostsPath, data, 0644)

	restoreAttributes(origAttrs)

	if writeErr != nil {
		return fmt.Errorf("写入 hosts 失败: %w", writeErr)
	}
	fmt.Printf("已从备份恢复 hosts\n")

	// 5. 刷新 DNS
	_ = exec.Command("ipconfig", "/flushdns").Run()
	fmt.Println("已刷新 DNS 缓存。")

	return nil
}
