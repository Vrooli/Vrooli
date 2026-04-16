Param(
    [string]$ModulePath = "resources/neo4j/cli",
    [string]$Name = "resource-neo4j",
    [string]$Manifest = "resources/neo4j/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
