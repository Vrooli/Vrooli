Param(
    [string]$ModulePath = "scenarios/test-data-generator/cli",
    [string]$Name = "test-data-generator",
    [string]$Manifest = "scenarios/test-data-generator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
