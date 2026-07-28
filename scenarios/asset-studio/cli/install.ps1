Param(
    [string]$ModulePath = "scenarios/asset-studio/cli",
    [string]$Name = "asset-studio",
    [string]$Manifest = "scenarios/asset-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
