Param(
    [string]$ModulePath = "scenarios/chart-generator/cli",
    [string]$Name = "chart-generator",
    [string]$Manifest = "scenarios/chart-generator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
