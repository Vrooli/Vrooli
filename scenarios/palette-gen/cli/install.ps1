Param(
    [string]$ModulePath = "scenarios/palette-gen/cli",
    [string]$Name = "palette-gen",
    [string]$Manifest = "scenarios/palette-gen/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
