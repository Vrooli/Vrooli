Param(
    [string]$ModulePath = "resources/litellm/cli",
    [string]$Name = "resource-litellm",
    [string]$Manifest = "resources/litellm/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
