Param(
    [string]$ModulePath = "scenarios/image-tools/cli",
    [string]$Name = "image-tools",
    [string]$Manifest = "scenarios/image-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
