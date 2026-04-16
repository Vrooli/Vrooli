Param(
    [string]$ModulePath = "scenarios/stream-of-consciousness-analyzer/cli",
    [string]$Name = "stream-of-consciousness-analyzer",
    [string]$Manifest = "scenarios/stream-of-consciousness-analyzer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
