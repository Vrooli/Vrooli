Param(
    [string]$ModulePath = "scenarios/treasury/cli",
    [string]$Name = "treasury",
    [string]$Manifest = "scenarios/treasury/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
