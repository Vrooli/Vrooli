Param(
    [string]$ModulePath = "scenarios/algorithm-library/cli",
    [string]$Name = "algorithm-library",
    [string]$Manifest = "scenarios/algorithm-library/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
