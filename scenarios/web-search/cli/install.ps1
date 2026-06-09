Param(
    [string]$ModulePath = "scenarios/web-search/cli",
    [string]$Name = "web-search",
    [string]$Manifest = "scenarios/web-search/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
