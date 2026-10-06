package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ggt/internal/admin"
	"ggt/internal/download"
	"ggt/internal/hosts"
	"ggt/internal/probe"
	"ggt/internal/proxy"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

const topN = 10

func main() {

	// 1. --version / -v：打印版本，无需管理员，直接返回
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("GGT v%s (built %s)\n", Version, BuildTime)
		return
	}

	// 先检查管理员权限（--restore 也需要）
	if !admin.IsAdmin2() {
		fmt.Println("== Google Translate Fix (By 52pojie fxcfasd) ==")
		fmt.Println("本程序需要管理员权限才能修改 hosts 文件。")
		fmt.Print("是否以管理员身份重启？(y/N): ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "y" || line == "Y" {
			if err := admin.RelaunchAsAdmin(); err != nil {
				fmt.Printf("提权失败: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		fmt.Println("已取消。")
		os.Exit(1)
	}

	// 再处理 --restore
	if len(os.Args) > 1 && (os.Args[1] == "--restore" || os.Args[1] == "-r") {
		fmt.Println("== Google Translate Fix (Go) ==")
		fmt.Println("正在从最新备份恢复 hosts ...")
		if err := hosts.Restore(); err != nil {
			fmt.Printf("恢复失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n完成！hosts 已恢复。")
		fmt.Println("按回车键退出...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	fmt.Println("== Google Translate Fix (By 52pojie fxcfasd) ==")

	// 检查管理员权限
	if !admin.IsAdmin2() {
		fmt.Println("本程序需要管理员权限才能修改 hosts 文件。")
		fmt.Print("是否以管理员身份重启？(y/N): ")

		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if line == "y" || line == "Y" {
			if err := admin.RelaunchAsAdmin(); err != nil {
				fmt.Printf("提权失败: %v\n", err)
				fmt.Println("请手动右键 → 以管理员身份运行。")
				os.Exit(1)
			}
			// 当前进程退出，新进程已经起来
			os.Exit(0)
		}

		fmt.Println("已取消。")
		os.Exit(1)
	}

	// 1. 下载 IP 列表（直连优先，失败后询问代理）
	body, err := fetchWithFallback()
	if err != nil {
		fmt.Printf("获取 IP 列表失败: %v\n", err)
		os.Exit(2)
	}

	// 2. 解析 IP 列表
	ips := parseIPList(string(body))
	if len(ips) == 0 {
		fmt.Println("没有获取到有效 IP 列表，退出。")
		os.Exit(2)
	}
	fmt.Printf("获取到 %d 个候选 IP。\n", len(ips))

	// 3. 并发探测
	fmt.Println("开始测速（TCP 443，直连）...")
	const maxProbe = 0 // 想测全部就填 0
	results := probe.ProbeAll(ips, maxProbe)
	if len(results) == 0 {
		fmt.Println("没有可用 IP，退出。")
		os.Exit(3)
	}

	// 4. 展示前 N
	n := topN
	if len(results) < n {
		n = len(results)
	}
	fmt.Println("\n序号 IP                TCP443(ms)")
	fmt.Println("--------------------------------------")
	for i := 0; i < n; i++ {
		fmt.Printf("%-4d %-17s %d\n", i+1, results[i].IP, results[i].TCPMs)
	}

	// 5. 用户选择
	chosen := askIndex(n)
	if chosen < 0 {
		fmt.Println("已取消。")
		return
	}
	ip := results[chosen].IP

	// 6. 修改 hosts
	if err := hosts.Update(ip); err != nil {
		fmt.Printf("修改 hosts 失败: %v\n", err)
		os.Exit(5)
	}

	fmt.Printf("\n完成！已将 Google 翻译域名指向 %s\n", ip)
	fmt.Println("按回车键退出...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// fetchWithFallback 折中方案：环境变量代理 -> 直连 -> 询问
func fetchWithFallback() ([]byte, error) {
	// 1. 环境变量代理
	if p := proxy.FromEnv(); p != nil {
		fmt.Printf("检测到环境变量代理: %s\n", p)
		body, err := download.Fetch(p)
		if err == nil {
			return body, nil
		}
		fmt.Printf("环境变量代理下载失败: %v\n", err)
	}

	// 2. 直连
	fmt.Println("尝试直连下载 IP 列表...")
	body, err := download.Fetch(nil)
	if err == nil {
		return body, nil
	}
	fmt.Printf("直连下载失败: %v\n", err)

	// 3. 询问用户
	fmt.Print("是否使用代理重试？(y/N): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "y" && line != "Y" {
		return nil, fmt.Errorf("用户取消")
	}

	p := proxy.Ask()
	return download.Fetch(p)
}

// parseIPList 解析、去重、校验 IPv4
func parseIPList(raw string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 取第一个 token
		fields := strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ','
		})
		if len(fields) == 0 {
			continue
		}
		ip := fields[0]
		if !isIPv4(ip) {
			continue
		}
		if seen[ip] {
			continue
		}
		seen[ip] = true
		out = append(out, ip)
	}
	return out
}

func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

// askIndex 询问序号，返回 0-based 索引，-1 表示取消
func askIndex(max int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("请输入序号 (1-%d)，直接回车取消: ", max)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return -1
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > max {
			fmt.Println("输入无效，请重新输入。")
			continue
		}
		return n - 1
	}
}
