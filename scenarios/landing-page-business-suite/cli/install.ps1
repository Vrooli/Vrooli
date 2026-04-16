Param(
    [string]$ModulePath = "scenarios/landing-page-business-suite/cli",
    [string]$Name = "landing-page-business-suite",
    [string]$Manifest = "scenarios/landing-page-business-suite/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
