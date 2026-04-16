Param(
    [string]$ModulePath = "scenarios/browser-automation-studio/cli",
    [string]$Name = "browser-automation-studio",
    [string]$Manifest = "scenarios/browser-automation-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
