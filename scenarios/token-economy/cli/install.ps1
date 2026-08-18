Param(
    [string]$ModulePath = "scenarios/token-economy/cli",
    [string]$Name = "token-economy",
    [string]$Manifest = "scenarios/token-economy/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
