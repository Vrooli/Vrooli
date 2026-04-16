Param(
    [string]$ModulePath = "scenarios/symbol-search/cli",
    [string]$Name = "symbol-search",
    [string]$Manifest = "scenarios/symbol-search/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
