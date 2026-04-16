Param(
    [string]$ModulePath = "scenarios/git-control-tower/cli",
    [string]$Name = "git-control-tower",
    [string]$Manifest = "scenarios/git-control-tower/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
