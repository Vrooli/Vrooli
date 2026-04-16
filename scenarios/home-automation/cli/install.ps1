Param(
    [string]$ModulePath = "scenarios/home-automation/cli",
    [string]$Name = "home-automation",
    [string]$Manifest = "scenarios/home-automation/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
