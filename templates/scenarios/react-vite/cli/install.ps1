Param(
    [string]$ModulePath = "scenarios/{{SCENARIO_ID}}/cli",
    [string]$Name = "{{SCENARIO_ID}}",
    [string]$Manifest = "scenarios/{{SCENARIO_ID}}/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
