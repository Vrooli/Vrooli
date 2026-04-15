Param(
    [string]$ModulePath = "scenarios/browser-automation-studio/cli",
    [string]$Name = "browser-automation-studio"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
