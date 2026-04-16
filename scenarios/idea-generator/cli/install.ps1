Param(
    [string]$ModulePath = "scenarios/idea-generator/cli",
    [string]$Name = "idea-generator",
    [string]$Manifest = "scenarios/idea-generator/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
