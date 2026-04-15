Param(
    [string]$ModulePath = "scenarios/workspace-sandbox/cli",
    [string]$Name = "workspace-sandbox"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
