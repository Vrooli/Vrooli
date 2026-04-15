Param(
    [string]$ModulePath = "scenarios/knowledge-observatory/cli",
    [string]$Name = "knowledge-observatory"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
