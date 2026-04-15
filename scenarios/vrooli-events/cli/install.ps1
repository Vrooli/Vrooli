Param(
    [string]$ModulePath = "scenarios/vrooli-events/cli",
    [string]$Name = "vrooli-events"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
