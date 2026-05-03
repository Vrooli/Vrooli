Param(
    [string]$ModulePath = "scenarios/smoke-tier1/cli",
    [string]$Name = "smoke-tier1",
    [string]$Manifest = "scenarios/smoke-tier1/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
