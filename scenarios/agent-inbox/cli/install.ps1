Param(
    [string]$ModulePath = "scenarios/agent-inbox/cli",
    [string]$Name = "agent-inbox"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name
