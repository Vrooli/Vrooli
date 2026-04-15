Param(
    [string]$ModulePath = "scenarios/ecosystem-manager/cli",
    [string]$Name = "ecosystem-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
