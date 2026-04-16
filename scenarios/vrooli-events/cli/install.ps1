Param(
    [string]$ModulePath = "scenarios/vrooli-events/cli",
    [string]$Name = "vrooli-events",
    [string]$Manifest = "scenarios/vrooli-events/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
