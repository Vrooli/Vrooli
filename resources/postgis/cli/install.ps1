Param(
    [string]$ModulePath = "resources/postgis/cli",
    [string]$Name = "resource-postgis"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
