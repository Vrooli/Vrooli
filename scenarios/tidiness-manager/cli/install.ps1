Param(
    [string]$ModulePath = "scenarios/tidiness-manager/cli",
    [string]$Name = "tidiness-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name
