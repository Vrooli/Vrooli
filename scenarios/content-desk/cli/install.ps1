Param(
    [string]$ModulePath = "scenarios/content-desk/cli",
    [string]$Name = "content-desk",
    [string]$Manifest = "scenarios/content-desk/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
