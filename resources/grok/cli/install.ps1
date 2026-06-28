Param(
    [string]$ModulePath = "resources/grok/cli",
    [string]$Name = "resource-grok",
    [string]$Manifest = "resources/grok/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
