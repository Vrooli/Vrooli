Param(
    [string]$ModulePath = "scenarios/network-manager/cli",
    [string]$Name = "network-manager",
    [string]$Manifest = "scenarios/network-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
