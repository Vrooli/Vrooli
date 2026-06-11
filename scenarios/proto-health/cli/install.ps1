Param(
    [string]$ModulePath = "scenarios/proto-health/cli",
    [string]$Name = "proto-health",
    [string]$Manifest = "scenarios/proto-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
