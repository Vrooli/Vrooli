Param(
    [string]$ModulePath = "scenarios/hello-mobile/cli",
    [string]$Name = "hello-mobile",
    [string]$Manifest = "scenarios/hello-mobile/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
