Param(
    [string]$ModulePath = "scenarios/browser-automation-studio/cli",
    [string]$Name = "browser-automation-studio",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot
