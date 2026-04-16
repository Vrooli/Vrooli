Param(
    [string]$ModulePath = "scenarios/agent-metareasoning-manager/cli",
    [string]$Name = "agent-metareasoning-manager",
    [string]$Manifest = "scenarios/agent-metareasoning-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
