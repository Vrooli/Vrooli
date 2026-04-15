Param(
    [string]$ModulePath = "resources/opencode/cli",
    [string]$Name = "resource-opencode"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
