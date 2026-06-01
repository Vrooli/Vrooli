Param(
    [string]$ModulePath = "scenarios/security-health/cli",
    [string]$Name = "security-health",
    [string]$Manifest = "scenarios/security-health/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
