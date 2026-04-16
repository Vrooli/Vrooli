Param(
    [string]$ModulePath = "resources/kokoro/cli",
    [string]$Name = "resource-kokoro",
    [string]$Manifest = "resources/kokoro/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
