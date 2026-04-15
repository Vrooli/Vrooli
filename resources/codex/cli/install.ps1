Param(
    [string]$ModulePath = "resources/codex/cli",
    [string]$Name = "resource-codex"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
