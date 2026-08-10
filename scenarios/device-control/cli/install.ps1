Param(
    [string]$ModulePath = "scenarios/device-control/cli",
    [string]$Name = "device-control",
    [string]$Manifest = "scenarios/device-control/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
