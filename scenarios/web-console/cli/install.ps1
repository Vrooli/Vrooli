Param(
    [string]$ModulePath = "scenarios/web-console/cli",
    [string]$Name = "web-console"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
