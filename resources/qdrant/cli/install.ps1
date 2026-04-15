Param(
    [string]$ModulePath = "resources/qdrant/cli",
    [string]$Name = "resource-qdrant"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
