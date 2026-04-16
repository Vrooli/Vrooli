Param(
    [string]$ModulePath = "scenarios/data-backup-manager/cli",
    [string]$Name = "data-backup-manager",
    [string]$Manifest = "scenarios/data-backup-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
