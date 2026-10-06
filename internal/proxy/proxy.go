package proxy

import (
	"fmt"
	"os"
	"strings"
)

type Kind int

const (
	None Kind = iota
	HTTP
	SOCKS5
)

type Config struct {
	Kind Kind
	Host string
	Port int
}

func (c Config) String() string {
	switch c.Kind {
	case None:
		return "(不使用代理)"
	case HTTP:
		return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
	case SOCKS5:
		return fmt.Sprintf("socks5://%s:%d", c.Host, c.Port)
	}
	return "(未知)"
}

// FromEnv 从环境变量读取代理
func FromEnv() *Config {
	keys := []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY", "all_proxy", "ALL_PROXY"}
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		if cfg := parseURL(v); cfg != nil {
			return cfg
		}
	}
	return nil
}

func parseURL(url string) *Config {
	var kind Kind
	rest := url
	switch {
	case strings.HasPrefix(rest, "http://"):
		rest = rest[len("http://"):]
		kind = HTTP
	case strings.HasPrefix(rest, "https://"):
		rest = rest[len("https://"):]
		kind = HTTP
	case strings.HasPrefix(rest, "socks5h://"):
		rest = rest[len("socks5h://"):]
		kind = SOCKS5
	case strings.HasPrefix(rest, "socks5://"):
		rest = rest[len("socks5://"):]
		kind = SOCKS5
	default:
		return nil
	}

	// 去掉 user:pass@
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest = rest[i+1:]
	}
	// 去掉路径
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}

	i := strings.LastIndex(rest, ":")
	if i < 0 {
		return nil
	}
	host := rest[:i]
	var port int
	if _, err := fmt.Sscanf(rest[i+1:], "%d", &port); err != nil {
		return nil
	}
	return &Config{Kind: kind, Host: host, Port: port}
}

// Ask 交互式询问代理配置
func Ask() *Config {
	fmt.Print("请选择代理类型 [1] HTTP  [2] SOCKS5 (默认 1): ")
	var t string
	fmt.Scanln(&t)
	kind := HTTP
	if strings.TrimSpace(t) == "2" {
		kind = SOCKS5
	}

	fmt.Print("请输入代理地址 (默认 127.0.0.1): ")
	var host string
	fmt.Scanln(&host)
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}

	var port int
	for {
		fmt.Print("请输入代理端口 (1-65535): ")
		var p string
		fmt.Scanln(&p)
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &port); err == nil && port >= 1 && port <= 65535 {
			break
		}
		fmt.Println("端口无效，请重新输入。")
	}

	return &Config{Kind: kind, Host: host, Port: port}
}
