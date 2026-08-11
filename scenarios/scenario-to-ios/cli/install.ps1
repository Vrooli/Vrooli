Param(
    [string]$ModulePath = "scenarios/scenario-to-ios/cli",
    [string]$Name = "scenario-to-ios",
    [string]$Manifest = "scenarios/scenario-to-ios/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
