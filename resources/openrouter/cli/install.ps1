Param(
    [string]$ModulePath = "resources/openrouter/cli",
    [string]$Name = "resource-openrouter",
    [string]$Manifest = "resources/openrouter/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
