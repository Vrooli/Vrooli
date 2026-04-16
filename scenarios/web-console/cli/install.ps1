Param(
    [string]$ModulePath = "scenarios/web-console/cli",
    [string]$Name = "web-console",
    [string]$Manifest = "scenarios/web-console/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
