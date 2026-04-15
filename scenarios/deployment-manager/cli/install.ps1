Param(
    [string]$ModulePath = "scenarios/deployment-manager/cli",
    [string]$Name = "deployment-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
