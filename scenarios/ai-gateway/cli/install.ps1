Param(
    [string]$ModulePath = "scenarios/ai-gateway/cli",
    [string]$Name = "ai-gateway",
    [string]$Manifest = "scenarios/ai-gateway/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
