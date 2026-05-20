Param(
    [string]$ModulePath = "scenarios/ui-health/cli",
    [string]$Name = "ui-health",
    [string]$Manifest = "scenarios/ui-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
