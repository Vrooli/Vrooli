Param(
    [string]$ModulePath = "scenarios/notes/cli",
    [string]$Name = "notes",
    [string]$Manifest = "scenarios/notes/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
