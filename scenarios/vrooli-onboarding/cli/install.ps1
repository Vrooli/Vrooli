Param(
    [string]$ModulePath = "scenarios/vrooli-onboarding/cli",
    [string]$Name = "vrooli-onboarding",
    [string]$Manifest = "scenarios/vrooli-onboarding/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
