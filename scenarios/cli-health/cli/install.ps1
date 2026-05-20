Param(
    [string]$ModulePath = "scenarios/cli-health/cli",
    [string]$Name = "cli-health",
    [string]$Manifest = "scenarios/cli-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
