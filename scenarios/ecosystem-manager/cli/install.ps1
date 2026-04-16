Param(
    [string]$ModulePath = "scenarios/ecosystem-manager/cli",
    [string]$Name = "ecosystem-manager",
    [string]$Manifest = "scenarios/ecosystem-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
