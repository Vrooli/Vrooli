Param(
    [string]$ModulePath = "resources/questdb/cli",
    [string]$Name = "resource-questdb",
    [string]$Manifest = "resources/questdb/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
