Param(
    [string]$ModulePath = "scenarios/ai-chatbot-manager/cli",
    [string]$Name = "ai-chatbot-manager",
    [string]$Manifest = "scenarios/ai-chatbot-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
