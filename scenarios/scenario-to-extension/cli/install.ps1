Param(
    [string]$ModulePath = "scenarios/scenario-to-extension/cli",
    [string]$Name = "scenario-to-extension",
    [string]$Manifest = "scenarios/scenario-to-extension/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
