Param(
    [string]$ModulePath = "resources/cloudflare-ai-gateway/cli",
    [string]$Name = "resource-cloudflare-ai-gateway"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
