Param(
    [string]$ModulePath = "resources/home-assistant/cli",
    [string]$Name = "resource-home-assistant",
    [string]$Manifest = "resources/home-assistant/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
