Param(
    [string]$ModulePath = "resources/cloudflare-ai-gateway/cli",
    [string]$Name = "resource-cloudflare-ai-gateway",
    [string]$Manifest = "resources/cloudflare-ai-gateway/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
