Param(
    [string]$ModulePath = "scenarios/vrooli-orchestrator/cli",
    [string]$Name = "vrooli-orchestrator",
    [string]$Manifest = "scenarios/vrooli-orchestrator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
