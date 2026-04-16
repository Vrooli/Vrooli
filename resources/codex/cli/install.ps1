Param(
    [string]$ModulePath = "resources/codex/cli",
    [string]$Name = "resource-codex",
    [string]$Manifest = "resources/codex/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
