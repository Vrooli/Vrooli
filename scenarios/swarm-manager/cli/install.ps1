Param(
    [string]$ModulePath = "scenarios/swarm-manager/cli",
    [string]$Name = "swarm-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
