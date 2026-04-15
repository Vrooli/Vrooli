Param(
    [string]$ModulePath = "scenarios/agent-manager/cli",
    [string]$Name = "agent-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
