Param(
    [string]$ModulePath = "scenarios/accessibility-compliance-hub/cli",
    [string]$Name = "accessibility-compliance-hub",
    [string]$Manifest = "scenarios/accessibility-compliance-hub/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
