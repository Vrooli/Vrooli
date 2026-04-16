Param(
    [string]$ModulePath = "scenarios/vrooli-autoheal/cli",
    [string]$Name = "vrooli-autoheal",
    [string]$Manifest = "scenarios/vrooli-autoheal/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
