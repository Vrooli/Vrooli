Param(
    [string]$ModulePath = "scenarios/workspace-sandbox/cli",
    [string]$Name = "workspace-sandbox",
    [string]$Manifest = "scenarios/workspace-sandbox/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
