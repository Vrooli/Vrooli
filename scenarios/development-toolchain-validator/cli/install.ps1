Param(
    [string]$ModulePath = "scenarios/development-toolchain-validator/cli",
    [string]$Name = "development-toolchain-validator",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot
