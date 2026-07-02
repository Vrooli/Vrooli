Param(
    [string]$ModulePath = "scenarios/business-health/cli",
    [string]$Name = "business-health",
    [string]$Manifest = "scenarios/business-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
