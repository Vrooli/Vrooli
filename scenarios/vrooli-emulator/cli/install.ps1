Param(
    [string]$ModulePath = "scenarios/vrooli-emulator/cli",
    [string]$Name = "vrooli-emulator",
    [string]$Manifest = "scenarios/vrooli-emulator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
