Param(
    [string]$ModulePath = "scenarios/reference-react-vite/cli",
    [string]$Name = "reference-react-vite",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot
