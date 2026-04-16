Param(
    [string]$ModulePath = "scenarios/scenario-to-android/cli",
    [string]$Name = "scenario-to-android",
    [string]$Manifest = "scenarios/scenario-to-android/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
