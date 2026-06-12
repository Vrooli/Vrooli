Param(
    [string]$ModulePath = "scenarios/code-facts/cli",
    [string]$Name = "code-facts",
    [string]$Manifest = "scenarios/code-facts/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
