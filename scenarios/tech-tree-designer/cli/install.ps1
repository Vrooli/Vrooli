Param(
    [string]$ModulePath = "scenarios/tech-tree-designer/cli",
    [string]$Name = "tech-tree-designer",
    [string]$Manifest = "scenarios/tech-tree-designer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
