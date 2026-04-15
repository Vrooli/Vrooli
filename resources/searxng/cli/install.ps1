Param(
    [string]$ModulePath = "resources/searxng/cli",
    [string]$Name = "resource-searxng"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
