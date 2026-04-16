Param(
    [string]$ModulePath = "scenarios/scenario-authenticator/cli",
    [string]$Name = "scenario-authenticator",
    [string]$Manifest = "scenarios/scenario-authenticator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
