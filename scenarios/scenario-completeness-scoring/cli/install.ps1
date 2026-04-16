Param(
    [string]$ModulePath = "scenarios/scenario-completeness-scoring/cli",
    [string]$Name = "scenario-completeness-scoring",
    [string]$Manifest = "scenarios/scenario-completeness-scoring/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
