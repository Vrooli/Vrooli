Param(
    [string]$ModulePath = "scenarios/vrooli-bridge/cli",
    [string]$Name = "vrooli-bridge",
    [string]$Manifest = "scenarios/vrooli-bridge/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
