Param(
    [string]$ModulePath = "scenarios/landing-page-business-suite/cli",
    [string]$Name = "landing-page-business-suite"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
