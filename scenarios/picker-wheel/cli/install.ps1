Param(
    [string]$ModulePath = "scenarios/picker-wheel/cli",
    [string]$Name = "picker-wheel",
    [string]$Manifest = "scenarios/picker-wheel/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
