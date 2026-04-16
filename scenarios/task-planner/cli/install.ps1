Param(
    [string]$ModulePath = "scenarios/task-planner/cli",
    [string]$Name = "task-planner",
    [string]$Manifest = "scenarios/task-planner/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
