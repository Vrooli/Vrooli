Param(
    [string]$ModulePath = "resources/postgres/cli",
    [string]$Name = "resource-postgres",
    [string]$Manifest = "resources/postgres/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
