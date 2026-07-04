Param(
    [string]$ModulePath = "scenarios/experience-manager/cli",
    [string]$Name = "experience-manager",
    [string]$Manifest = "scenarios/experience-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
