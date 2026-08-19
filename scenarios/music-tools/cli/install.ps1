Param(
    [string]$ModulePath = "scenarios/music-tools/cli",
    [string]$Name = "music-tools",
    [string]$Manifest = "scenarios/music-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
