Param(
    [string]$ModulePath = "scenarios/campaign-content-studio/cli",
    [string]$Name = "campaign-content-studio",
    [string]$Manifest = "scenarios/campaign-content-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
