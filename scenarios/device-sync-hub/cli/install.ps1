Param(
    [string]$ModulePath = "scenarios/device-sync-hub/cli",
    [string]$Name = "device-sync-hub",
    [string]$Manifest = "scenarios/device-sync-hub/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
