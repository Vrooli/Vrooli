Param(
    [string]$ModulePath = "scenarios/development-toolchain-validator/cli",
    [string]$Name = "development-toolchain-validator"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
