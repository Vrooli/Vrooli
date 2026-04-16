Param(
    [string]$ModulePath = "scenarios/calendar/cli",
    [string]$Name = "calendar",
    [string]$Manifest = "scenarios/calendar/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
