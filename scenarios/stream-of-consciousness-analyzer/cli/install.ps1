Param(
    [string]$ModulePath = "scenarios/stream-of-consciousness-analyzer/cli",
    [string]$Name = "stream-of-consciousness-analyzer",
    [string]$AppRoot = ""
)

if (-not $AppRoot -or $AppRoot -eq "") {
    $AppRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path
}

& "$AppRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -AppRoot $AppRoot
