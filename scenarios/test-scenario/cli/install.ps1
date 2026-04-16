Param(
    [string]$ModulePath = "scenarios/test-scenario/cli",
    [string]$Name = "test-scenario",
    [string]$Manifest = "scenarios/test-scenario/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
