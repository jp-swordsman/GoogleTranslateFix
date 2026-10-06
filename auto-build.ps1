# build.ps1 - 一次生成 64 位和 32 位两个版本的 GGT.exe
#
# 用法:
#   .\build.ps1                    # 用默认版本号
#   .\build.ps1 -Version 1.2.3     # 指定版本号
#   .\build.ps1 -Clean             # 先清理再编译
#   .\build.ps1 -UPX               # 编译后用 UPX 压缩（需要装 upx）
#   .\build.ps1 -Only x64          # 只编译 64 位
#   .\build.ps1 -Only x86          # 只编译 32 位

param(
    [string]$Version = "1.0.0",
    [switch]$Clean,
    [switch]$UPX,
    [ValidateSet("x64", "x86", "both")]
    [string]$Only = "both"
)

$ErrorActionPreference = "Stop"

# ---------- 颜色输出助手 ----------
function Write-Info    { param($msg) Write-Host $msg -ForegroundColor Cyan }
function Write-OK      { param($msg) Write-Host $msg -ForegroundColor Green }
function Write-Warn    { param($msg) Write-Host $msg -ForegroundColor Yellow }
function Write-Fail    { param($msg) Write-Host $msg -ForegroundColor Red }

# ---------- 进入脚本所在目录 ----------
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

Write-Info "== GGT Build Script =="
Write-Info "目录: $scriptDir"
Write-Info "版本: $Version"
Write-Info ""

# ---------- 清理 ----------
if ($Clean) {
    Write-Info "清理旧的构建产物 ..."
    Remove-Item -Force -ErrorAction SilentlyContinue GGT-x64.exe
    Remove-Item -Force -ErrorAction SilentlyContinue GGT-x86.exe
    Remove-Item -Force -ErrorAction SilentlyContinue GGT-x64-upx.exe
    Remove-Item -Force -ErrorAction SilentlyContinue GGT-x86-upx.exe
    Write-OK "清理完成。"
    Write-Info ""
}

# ---------- 检查 rsrc.syso（图标） ----------
if (Test-Path "GGT.ico") {
    if (-not (Test-Path "rsrc_windows_amd64.syso") -or -not (Test-Path "rsrc_windows_386.syso")) {
        Write-Warn "GGT.ico found but syso missing, generating ..."
        if (Get-Command rsrc -ErrorAction SilentlyContinue) {
            rsrc -ico GGT.ico -arch amd64 -o rsrc_windows_amd64.syso
            rsrc -ico GGT.ico -arch 386   -o rsrc_windows_386.syso
            Write-OK "syso generated for amd64 and 386."
        } else {
            Write-Warn "rsrc not found."
        }
    } else {
        Write-Info "syso exists."
    }
    Write-Info ""
}

# ---------- 公共环境变量 ----------
$env:CGO_ENABLED = "0"
$buildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ss"
$ldflags = "-s -w -X main.Version=$Version -X main.BuildTime=$buildTime"

# ---------- 编译函数 ----------
function Build-Target {
    param(
        [string]$Arch,      # amd64 或 386
        [string]$Output
    )

    Write-Info "编译 $Output (GOARCH=$Arch) ..."
    $env:GOARCH = $Arch
    $env:GOOS   = "windows"

        go build -ldflags="$ldflags" -o $Output .
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "编译失败: $Output"
        return $false
    }

    $size = [math]::Round((Get-Item $Output).Length / 1MB, 2)
    Write-OK "  -> $Output ($size MB)"
    return $true
}

# ---------- UPX 压缩函数 ----------
function Compress-UPX {
    param([string]$InputFile)

    if (-not $UPX) { return }

    if (-not (Get-Command upx -ErrorAction SilentlyContinue)) {
        Write-Warn "未找到 upx，跳过压缩。安装: winget install upx.upx"
        return
    }

    Write-Info "UPX 压缩 $InputFile ..."
    upx --best --lzma --no-encrypt $InputFile | Out-Null
    $size = [math]::Round((Get-Item $InputFile).Length / 1MB, 2)
    Write-OK "  -> $InputFile ($size MB)"
}

# ---------- 开始编译 ----------
$success = $true

if ($Only -eq "x64" -or $Only -eq "both") {
    $ok = Build-Target -Arch "amd64" -Output "GGT-x64.exe"
    if ($ok) { Compress-UPX "GGT-x64.exe" } else { $success = $false }
}

if ($Only -eq "x86" -or $Only -eq "both") {
    $ok = Build-Target -Arch "386" -Output "GGT-x86.exe"
    if ($ok) { Compress-UPX "GGT-x86.exe" } else { $success = $false }
}

# ---------- 总结 ----------
Write-Info ""
if ($success) {
    Write-OK "== 构建完成 =="
    Get-ChildItem GGT-*.exe | ForEach-Object {
        $size = [math]::Round($_.Length / 1MB, 2)
        Write-Host ("  {0,-20} {1,6} MB" -f $_.Name, $size) -ForegroundColor White
    }
} else {
    Write-Fail "== 构建失败，请检查上面的错误 =="
    exit 1
}

# 恢复环境变量
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:GOOS   -ErrorAction SilentlyContinue

# ---------- 使用提示 ----------
Write-Info ""
Write-Info "使用示例:"
Write-Info "  GGT-x64.exe              # 正常运行（自动请求管理员）"
Write-Info "  GGT-x64.exe --version    # 查看版本"
Write-Info "  GGT-x64.exe --restore    # 恢复 hosts"