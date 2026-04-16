Param(
    [string]$ModulePath = "scenarios/visited-tracker/cli",
    [string]$Name = "visited-tracker",
    [string]$Manifest = "scenarios/visited-tracker/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
