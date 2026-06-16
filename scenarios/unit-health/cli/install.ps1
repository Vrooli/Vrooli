Param(
    [string]$ModulePath = "scenarios/unit-health/cli",
    [string]$Name = "unit-health",
    [string]$Manifest = "scenarios/unit-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
