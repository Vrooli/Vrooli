Param(
    [string]$ModulePath = "scenarios/fall-foliage-explorer/cli",
    [string]$Name = "fall-foliage-explorer",
    [string]$Manifest = "scenarios/fall-foliage-explorer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
