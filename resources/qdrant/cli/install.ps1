Param(
    [string]$ModulePath = "resources/qdrant/cli",
    [string]$Name = "resource-qdrant",
    [string]$Manifest = "resources/qdrant/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
