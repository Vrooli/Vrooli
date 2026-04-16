Param(
    [string]$ModulePath = "scenarios/bedtime-story-generator/cli",
    [string]$Name = "bedtime-story-generator",
    [string]$Manifest = "scenarios/bedtime-story-generator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
