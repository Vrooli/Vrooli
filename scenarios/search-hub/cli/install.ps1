Param(
    [string]$ModulePath = "scenarios/search-hub/cli",
    [string]$Name = "search-hub",
    [string]$Manifest = "scenarios/search-hub/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
