param()

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../..")).Path

& "$repoRoot\packages\cli-core\install.ps1" "scenarios/prompt-manager/cli" -Name "prompt-manager"
