Param(
    [string]$ModulePath = "scenarios/audio-tools/cli",
    [string]$Name = "audio-tools",
    [string]$Manifest = "scenarios/audio-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
