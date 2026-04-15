Param(
    [string]$ModulePath = "scenarios/scenario-to-desktop/cli",
    [string]$Name = "scenario-to-desktop"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
