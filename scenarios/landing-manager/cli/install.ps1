Param(
    [string]$ModulePath = "scenarios/landing-manager/cli",
    [string]$Name = "landing-manager",
    [string]$Manifest = "scenarios/landing-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
