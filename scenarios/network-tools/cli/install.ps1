Param(
    [string]$ModulePath = "scenarios/network-tools/cli",
    [string]$Name = "network-tools",
    [string]$Manifest = "scenarios/network-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
