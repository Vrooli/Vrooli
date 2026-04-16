Param(
    [string]$ModulePath = "resources/whisper/cli",
    [string]$Name = "resource-whisper",
    [string]$Manifest = "resources/whisper/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
