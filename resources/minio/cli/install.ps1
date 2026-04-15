Param(
    [string]$ModulePath = "resources/minio/cli",
    [string]$Name = "resource-minio"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
