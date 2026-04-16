Param(
    [string]$ModulePath = "scenarios/tidiness-manager/cli",
    [string]$Name = "tidiness-manager",
    [string]$Manifest = "scenarios/tidiness-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
