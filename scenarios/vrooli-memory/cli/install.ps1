Param(
    [string]$ModulePath = "scenarios/vrooli-memory/cli",
    [string]$Name = "vrooli-memory",
    [string]$Manifest = "scenarios/vrooli-memory/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
