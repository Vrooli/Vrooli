Param(
    [string]$ModulePath = "scenarios/llm-evaluator/cli",
    [string]$Name = "llm-evaluator"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
