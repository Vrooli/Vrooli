Param(
    [string]$ModulePath = "scenarios/seo-optimizer/cli",
    [string]$Name = "seo-optimizer",
    [string]$Manifest = "scenarios/seo-optimizer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
