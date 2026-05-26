Param(
    [string]$ModulePath = "resources/kopia/cli",
    [string]$Name = "resource-kopia",
    [string]$Manifest = "resources/kopia/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
