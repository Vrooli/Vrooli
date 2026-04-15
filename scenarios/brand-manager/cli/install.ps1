Param(
    [string]$ModulePath = "scenarios/brand-manager/cli",
    [string]$Name = "brand-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
