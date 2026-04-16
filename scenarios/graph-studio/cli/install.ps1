Param(
    [string]$ModulePath = "scenarios/graph-studio/cli",
    [string]$Name = "graph-studio",
    [string]$Manifest = "scenarios/graph-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
