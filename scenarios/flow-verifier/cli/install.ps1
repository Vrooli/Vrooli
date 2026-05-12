Param(
    [string]$ModulePath = "scenarios/flow-verifier/cli",
    [string]$Name = "flow-verifier",
    [string]$Manifest = "scenarios/flow-verifier/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
