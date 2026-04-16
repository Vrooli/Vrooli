Param(
    [string]$ModulePath = "resources/k6/cli",
    [string]$Name = "resource-k6",
    [string]$Manifest = "resources/k6/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
