Param(
    [string]$ModulePath = "resources/cloudflare-ai-gateway/cli",
    [string]$Name = "resource-cloudflare-ai-gateway",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot
