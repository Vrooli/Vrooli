Param(
    [string]$ModulePath = "scenarios/api-health/cli",
    [string]$Name = "api-health",
    [string]$Manifest = "scenarios/api-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
