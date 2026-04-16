Param(
    [string]$ModulePath = "scenarios/app-issue-tracker/cli",
    [string]$Name = "app-issue-tracker",
    [string]$Manifest = "scenarios/app-issue-tracker/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
