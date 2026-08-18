Param(
    [string]$ModulePath = "scenarios/persona/cli",
    [string]$Name = "persona",
    [string]$Manifest = "scenarios/persona/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
