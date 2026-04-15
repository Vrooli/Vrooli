Param(
    [string]$ModulePath = "resources/neo4j/cli",
    [string]$Name = "resource-neo4j"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
