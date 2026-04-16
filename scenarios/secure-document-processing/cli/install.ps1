Param(
    [string]$ModulePath = "scenarios/secure-document-processing/cli",
    [string]$Name = "secure-document-processing",
    [string]$Manifest = "scenarios/secure-document-processing/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
