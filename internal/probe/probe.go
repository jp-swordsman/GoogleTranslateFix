package probe

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	tcpTimeout  = 2 * time.Second
	concurrency = 64
)

type Result struct {
	IP    string
	TCPMs int64
}

// ProbeAll 并发测 TCP 443，返回所有能连上的 IP，按延迟升序。
// maxProbe 限制最多探测多少个 IP（0 表示全部）。
func ProbeAll(ips []string, maxProbe int) []Result {
	if maxProbe > 0 && len(ips) > maxProbe {
		ips = ips[:maxProbe]
	}
	total := len(ips)
	fmt.Printf("  共 %d 个候选，将测试 %d 个（并发 %d）\n", total, total, concurrency)

	var (
		mu        sync.Mutex
		results   []Result
		wg        sync.WaitGroup
		sem       = make(chan struct{}, concurrency)
		completed atomic.Int32
		stopCh    = make(chan struct{})
	)

	// 进度打印 goroutine
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				done := completed.Load()
				mu.Lock()
				found := len(results)
				mu.Unlock()
				fmt.Printf("\r  进度: %d / %d，已找到 %d 个可用 IP", done, total, found)
			}
		}
	}()

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer completed.Add(1)

			ms, err := tcpProbe(ip, 443, tcpTimeout)
			if err != nil {
				return
			}
			mu.Lock()
			results = append(results, Result{IP: ip, TCPMs: ms})
			mu.Unlock()
		}(ip)
	}
	wg.Wait()
	close(stopCh)

	// 换行，避免进度条和后续输出混在一起
	fmt.Println()

	sort.Slice(results, func(i, j int) bool {
		return results[i].TCPMs < results[j].TCPMs
	})
	return results
}

func tcpProbe(ip string, port int, timeout time.Duration) (int64, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return time.Since(start).Milliseconds(), nil
}
