Param(
    [string]$ModulePath = "scenarios/portal/cli",
    [string]$Name = "portal",
    [string]$Manifest = "scenarios/portal/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
