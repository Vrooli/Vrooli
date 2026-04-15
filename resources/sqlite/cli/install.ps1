Param(
    [string]$ModulePath = "resources/sqlite/cli",
    [string]$Name = "resource-sqlite"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
