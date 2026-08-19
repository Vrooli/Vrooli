Param(
    [string]$ModulePath = "scenarios/infrastructure-manager/cli",
    [string]$Name = "infrastructure-manager",
    [string]$Manifest = "scenarios/infrastructure-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
