Param(
    [string]$ModulePath = "scenarios/product-manager-agent/cli",
    [string]$Name = "product-manager-agent",
    [string]$Manifest = "scenarios/product-manager-agent/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
