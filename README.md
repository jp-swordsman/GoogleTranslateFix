# Google Translate Fix (GGT)

[![Release](https://img.shields.io/github/v/release/jp-swordsman/GoogleTranslateFix)](https://github.com/jp-swordsman/GoogleTranslateFix/releases)
[![Stars](https://img.shields.io/github/stars/jp-swordsman/GoogleTranslateFix?style=social)](https://github.com/jp-swordsman/GoogleTranslateFix/stargazers)
[![License](https://img.shields.io/github/license/jp-swordsman/GoogleTranslateFix)](https://github.com/jp-swordsman/GoogleTranslateFix/blob/main/LICENSE)

一键修复 Google 翻译无法访问的小工具。自动从多个 GitHub 镜像下载可用 IP 列表，并发测速筛选出最快的节点，写入系统 hosts 文件，让 `translate.google.com` / `translate.google.cn` 等域名直连可用。

**支持 Windows 10 / 11，64 位和 32 位。**

---

## ✨ 功能特点

- **自动测速**：并发 TCP 443 探测，从 8000+ 候选 IP 中筛出可用的，按延迟排序
- **多镜像回退**：内置 5 个 GitHub 加速镜像，某个挂了自动切下一个
- **代理支持**：直连失败时可选择使用 HTTP / SOCKS5 代理下载 IP 列表
- **一键修改 hosts**：备份原文件、去掉只读属性、写入新 IP、刷新 DNS，全程自动化
- **可恢复**：随时执行 `--restore` 从备份恢复原始 hosts
- **自动提权**：检测到非管理员时提示一键提权重启
- **单文件、无依赖**：编译产物是单个 exe，拷到任何 Windows 机器都能跑

---

## 📥 下载

从 [Releases](../../releases) 页面下载最新版本：

| 文件 | 适用系统 |
|------|---------|
| `GGT-x64.exe` | 64 位 Windows（推荐） |
| `GGT-x86.exe` | 32 位 Windows（64 位也能跑） |

---

## 🚀 使用方法

### 首次使用

1. **右键 → 以管理员身份运行**（或直接双击，程序会提示一键提权）
2. 等待程序下载 IP 列表并测速（约 1-2 分钟）
3. 从结果里选择一个延迟最低的 IP（一般选第 1 个）
4. 程序自动修改 hosts 并刷新 DNS
5. 打开浏览器访问 https://translate.google.cn 测试

### 恢复原始 hosts

如果想把 hosts 改回原样：

```powershell
GGT-x64.exe --restore
```

程序会从最新的备份文件恢复。

### 查看版本

```powershell
GGT-x64.exe --version
```

### 使用代理（可选）

如果下载 IP 列表失败，程序会提示是否使用代理。选择后输入：

- **类型**：HTTP 或 SOCKS5
- **地址**：默认 `127.0.0.1`
- **端口**：你的代理端口（如 Clash 默认 `7890`）

也可以提前设好环境变量，程序会自动读取：

```powershell
$env:https_proxy = "http://127.0.0.1:7890"
.\GGT-x64.exe
```

---

## 🔧 常见问题

### 1. 修改 hosts 后 Google 翻译还是打不开？

- 确认 hosts 文件确实被修改：`notepad C:\Windows\System32\drivers\etc\hosts`
- 确认已刷新 DNS：`ipconfig /flushdns`
- 尝试重启浏览器或换一个浏览器
- 如果 IP 本身失效，重新跑一次 `GGT-x64.exe` 换一个 IP

### 2. 提示"没有可用 IP"？

你当前的网络环境可能屏蔽了所有 Google 节点。可以：

- 换一个网络环境（如手机热点）再试
- 使用代理下载 IP 列表后，**探测仍然直连**（探测结果反映你的真实直连质量）

### 3. 杀毒软件报警？

工具会修改 hosts 文件，**某些杀毒软件可能误报**。请添加信任或临时关闭。

### 4. hosts 修改失败，提示"Access is denied"？

- 确认以**管理员身份**运行
- 检查 hosts 是否被其他软件锁定（如安全软件的"hosts 保护"）
- 手动去掉只读属性：`attrib -R C:\Windows\System32\drivers\etc\hosts`

### 5. 探测速度慢？

程序默认只测**前 3000 个 IP**，避免耗时过长。如需测全部，可修改源码 `internal/probe/probe.go` 里的 `maxProbe` 参数后重新编译。

---

## 🛠️ 从源码编译

### 环境要求

- Go 1.21+
- （可选）`rsrc` 用于嵌入图标：`go install github.com/akavel/rsrc@latest`

### 编译

```powershell
# 一键编译 64 位 + 32 位
.\build.ps1

# 指定版本号
.\build.ps1 -Version 1.0.0

# 先清理再编译
.\build.ps1 -Clean

# 用 UPX 压缩（需先装 upx）
.\build.ps1 -UPX
```

### 单独编译

```powershell
# 64 位
$env:CGO_ENABLED="0"; $env:GOARCH="amd64"
go build -ldflags="-s -w" -o GGT-x64.exe .

# 32 位
$env:CGO_ENABLED="0"; $env:GOARCH="386"
go build -ldflags="-s -w" -o GGT-x86.exe .
```

---

## 📁 项目结构

```
GGT/
├── main.go                      # 主流程
├── internal/
│   ├── admin/admin.go           # 管理员权限检测、提权重启
│   ├── download/download.go     # 多镜像下载 IP 列表
│   ├── probe/probe.go           # 并发 TCP 443 探测
│   ├── hosts/hosts.go           # hosts 备份、修改、恢复
│   └── proxy/proxy.go           # 代理配置、环境变量解析
├── build.ps1                    # 一键编译脚本
├── GGT.ico                      # 程序图标
├── rsrc_windows_amd64.syso      # 64 位图标资源
└── rsrc_windows_386.syso        # 32 位图标资源
```

---

## ⚙️ 工作原理

1. **下载 IP 列表**：从 GitHub 上的 [GoogleTranslateIpCheck](https://github.com/Ponderfly/GoogleTranslateIpCheck) 项目获取候选 IP，走多个加速镜像，任一成功即返回
2. **并发探测**：64 并发对每个 IP 的 443 端口做 TCP 连接测试，记录耗时
3. **排序筛选**：按延迟升序排列，取前 10 个展示
4. **修改 hosts**：备份原文件 → 去掉只读属性 → 删除旧的 `translate.google*` 行 → 追加新 IP → 刷新 DNS

**代理只用于第 1 步**（下载 IP 列表），探测阶段永远直连，确保测出的延迟反映真实网络质量。

---

## ⚠️ 免责声明

本工具仅用于学习和网络诊断，使用者需自行承担因修改 hosts 文件带来的风险。请确保遵守当地法律法规。

---

## 📄 许可证

MIT License

---

## 🙏 致谢

- [Ponderfly/GoogleTranslateIpCheck](https://github.com/Ponderfly/GoogleTranslateIpCheck) —— IP 列表来源