Param(
    [string]$ModulePath = "",
    [string]$Name = "",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

if (-not $ModulePath -or $ModulePath -eq "") {
    $ModulePath = (Resolve-Path $PSScriptRoot | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot

