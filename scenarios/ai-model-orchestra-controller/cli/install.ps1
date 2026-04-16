Param(
    [string]$ModulePath = "scenarios/ai-model-orchestra-controller/cli",
    [string]$Name = "ai-model-orchestra-controller",
    [string]$Manifest = "scenarios/ai-model-orchestra-controller/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
