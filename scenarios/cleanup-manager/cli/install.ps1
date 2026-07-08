Param(
    [string]$ModulePath = "scenarios/cleanup-manager/cli",
    [string]$Name = "cleanup-manager",
    [string]$Manifest = "scenarios/cleanup-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
