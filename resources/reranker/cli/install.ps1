Param(
    [string]$ModulePath = "resources/reranker/cli",
    [string]$Name = "resource-reranker",
    [string]$Manifest = "resources/reranker/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
