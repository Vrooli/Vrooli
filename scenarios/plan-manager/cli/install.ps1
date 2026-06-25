Param(
    [string]$ModulePath = "scenarios/plan-manager/cli",
    [string]$Name = "plan-manager",
    [string]$Manifest = "scenarios/plan-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
