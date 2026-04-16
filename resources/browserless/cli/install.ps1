Param(
    [string]$ModulePath = "resources/browserless/cli",
    [string]$Name = "resource-browserless",
    [string]$Manifest = "resources/browserless/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
