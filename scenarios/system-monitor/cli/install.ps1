Param(
    [string]$ModulePath = "scenarios/system-monitor/cli",
    [string]$Name = "system-monitor",
    [string]$Manifest = "scenarios/system-monitor/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
