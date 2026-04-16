Param(
    [string]$ModulePath = "scenarios/social-media-scheduler/cli",
    [string]$Name = "social-media-scheduler",
    [string]$Manifest = "scenarios/social-media-scheduler/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
