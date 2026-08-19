Param(
    [string]$ModulePath = "scenarios/music-library/cli",
    [string]$Name = "music-library",
    [string]$Manifest = "scenarios/music-library/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
