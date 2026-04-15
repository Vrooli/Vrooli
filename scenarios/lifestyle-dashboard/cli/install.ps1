Param(
    [string]$ModulePath = "scenarios/lifestyle-dashboard/cli",
    [string]$Name = "lifestyle-dashboard"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
