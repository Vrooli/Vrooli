Param(
    [string]$ModulePath = "scenarios/quality-health/cli",
    [string]$Name = "quality-health",
    [string]$Manifest = "scenarios/quality-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
