Param(
    [string]$ModulePath = "resources/kyutai-stt/cli",
    [string]$Name = "resource-kyutai-stt",
    [string]$Manifest = "resources/kyutai-stt/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
