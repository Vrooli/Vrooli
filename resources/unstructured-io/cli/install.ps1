Param(
    [string]$ModulePath = "resources/unstructured-io/cli",
    [string]$Name = "resource-unstructured-io",
    [string]$Manifest = "resources/unstructured-io/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
