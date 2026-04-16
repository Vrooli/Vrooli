Param(
    [string]$ModulePath = "scenarios/scalable-app-cookbook/cli",
    [string]$Name = "scalable-app-cookbook",
    [string]$Manifest = "scenarios/scalable-app-cookbook/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
