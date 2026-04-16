Param(
    [string]$ModulePath = "resources/opencode/cli",
    [string]$Name = "resource-opencode",
    [string]$Manifest = "resources/opencode/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
