Param(
    [string]$ModulePath = "scenarios/bookmark-intelligence-hub/cli",
    [string]$Name = "bookmark-intelligence-hub",
    [string]$Manifest = "scenarios/bookmark-intelligence-hub/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
