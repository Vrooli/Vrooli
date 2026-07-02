Param(
    [string]$ModulePath = "scenarios/workflow-health/cli",
    [string]$Name = "workflow-health",
    [string]$Manifest = "scenarios/workflow-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
