Param(
    [string]$ModulePath = "scenarios/app-monitor/cli",
    [string]$Name = "app-monitor",
    [string]$Manifest = "scenarios/app-monitor/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
