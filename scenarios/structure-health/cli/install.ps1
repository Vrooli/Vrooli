Param(
    [string]$ModulePath = "scenarios/structure-health/cli",
    [string]$Name = "structure-health",
    [string]$Manifest = "scenarios/structure-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
