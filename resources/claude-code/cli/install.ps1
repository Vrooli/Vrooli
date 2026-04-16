Param(
    [string]$ModulePath = "resources/claude-code/cli",
    [string]$Name = "resource-claude-code",
    [string]$Manifest = "resources/claude-code/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
