Param(
    [string]$ModulePath = "scenarios/vrooli-assistant/cli",
    [string]$Name = "vrooli-assistant",
    [string]$Manifest = "scenarios/vrooli-assistant/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
