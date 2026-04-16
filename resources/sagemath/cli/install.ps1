Param(
    [string]$ModulePath = "resources/sagemath/cli",
    [string]$Name = "resource-sagemath",
    [string]$Manifest = "resources/sagemath/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
