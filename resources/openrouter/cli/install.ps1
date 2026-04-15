Param(
    [string]$ModulePath = "resources/openrouter/cli",
    [string]$Name = "resource-openrouter"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
