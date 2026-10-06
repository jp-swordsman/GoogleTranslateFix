package download

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"ggt/internal/proxy"
)

var URLList = []string{
	"https://ghfast.top/https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
	"https://ghproxy.com/https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
	"https://gh-proxy.com/https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
	"https://cdn.jsdelivr.net/gh/Ponderfly/GoogleTranslateIpCheck@master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
	"https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
}

const (
	timeout     = 8 * time.Second
	maxBodySize = 8 * 1024 * 1024
)

// Fetch 下载 IP 列表。proxy == nil 表示直连。
func Fetch(p *proxy.Config) ([]byte, error) {
	client := buildClient(p)

	var lastErr error
	for _, u := range URLList {
		//fmt.Printf("  尝试下载: %s\n", u)
		body, err := fetchOne(client, u)
		if err != nil {
			//fmt.Printf("    失败: %v\n", err)
			lastErr = err
			continue
		}
		//fmt.Printf("    成功，收到 %d 字节\n", len(body))
		return body, nil
	}
	return nil, fmt.Errorf("全部 URL 失败: %w", lastErr)
}

func buildClient(p *proxy.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
	}

	if p != nil && p.Kind != proxy.None {
		var proxyURL *url.URL
		switch p.Kind {
		case proxy.HTTP:
			proxyURL, _ = url.Parse(fmt.Sprintf("http://%s:%d", p.Host, p.Port))
		case proxy.SOCKS5:
			proxyURL, _ = url.Parse(fmt.Sprintf("socks5://%s:%d", p.Host, p.Port))
		}
		if proxyURL != nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func fetchOne(client *http.Client, u string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GGT/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("空响应")
	}
	return body, nil
}
