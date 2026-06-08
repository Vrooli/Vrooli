Param(
    [string]$ModulePath = "scenarios/measures-health/cli",
    [string]$Name = "measures-health",
    [string]$Manifest = "scenarios/measures-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
