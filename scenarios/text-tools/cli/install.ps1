Param(
    [string]$ModulePath = "scenarios/text-tools/cli",
    [string]$Name = "text-tools",
    [string]$Manifest = "scenarios/text-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
