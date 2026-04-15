Param(
    [string]$ModulePath = "resources/ollama/cli",
    [string]$Name = "resource-ollama"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
