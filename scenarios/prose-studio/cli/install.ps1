Param(
    [string]$ModulePath = "scenarios/prose-studio/cli",
    [string]$Name = "prose-studio",
    [string]$Manifest = "scenarios/prose-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
