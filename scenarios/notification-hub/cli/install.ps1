Param(
    [string]$ModulePath = "scenarios/notification-hub/cli",
    [string]$Name = "notification-hub",
    [string]$Manifest = "scenarios/notification-hub/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
