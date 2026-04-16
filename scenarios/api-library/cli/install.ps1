Param(
    [string]$ModulePath = "scenarios/api-library/cli",
    [string]$Name = "api-library",
    [string]$Manifest = "scenarios/api-library/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
