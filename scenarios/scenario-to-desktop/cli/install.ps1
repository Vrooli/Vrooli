Param(
    [string]$ModulePath = "scenarios/scenario-to-desktop/cli",
    [string]$Name = "scenario-to-desktop",
    [string]$Manifest = "scenarios/scenario-to-desktop/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
