Param(
    [string]$ModulePath = "scenarios/channel-manager/cli",
    [string]$Name = "channel-manager",
    [string]$Manifest = "scenarios/channel-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
