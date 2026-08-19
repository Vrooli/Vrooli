Param(
    [string]$ModulePath = "scenarios/scenario-to-plugin/cli",
    [string]$Name = "scenario-to-plugin",
    [string]$Manifest = "scenarios/scenario-to-plugin/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
