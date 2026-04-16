Param(
    [string]$ModulePath = "scenarios/agent-inbox/cli",
    [string]$Name = "agent-inbox",
    [string]$Manifest = "scenarios/agent-inbox/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
