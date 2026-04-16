Param(
    [string]$ModulePath = "scenarios/secrets-manager/cli",
    [string]$Name = "secrets-manager",
    [string]$Manifest = "scenarios/secrets-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
