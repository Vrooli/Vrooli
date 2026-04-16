Param(
    [string]$ModulePath = "resources/minio/cli",
    [string]$Name = "resource-minio",
    [string]$Manifest = "resources/minio/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
