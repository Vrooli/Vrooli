Param(
    [string]$ModulePath = "scenarios/maintenance-orchestrator/cli",
    [string]$Name = "maintenance-orchestrator",
    [string]$Manifest = "scenarios/maintenance-orchestrator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
