Param(
    [string]$ModulePath = "scenarios/file-tools/cli",
    [string]$Name = "file-tools",
    [string]$Manifest = "scenarios/file-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
