Param(
    [string]$ModulePath = "resources/antigravity/cli",
    [string]$Name = "resource-antigravity",
    [string]$Manifest = "resources/antigravity/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
