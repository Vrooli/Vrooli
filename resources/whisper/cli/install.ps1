Param(
    [string]$ModulePath = "resources/whisper/cli",
    [string]$Name = "resource-whisper"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
