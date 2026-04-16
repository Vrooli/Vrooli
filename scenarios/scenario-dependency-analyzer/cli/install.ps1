Param(
    [string]$ModulePath = "scenarios/scenario-dependency-analyzer/cli",
    [string]$Name = "scenario-dependency-analyzer",
    [string]$Manifest = "scenarios/scenario-dependency-analyzer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
