param()

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
    $appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../.."))
}

& "$appRoot\packages\cli-core\install.ps1" "scenarios/prompt-manager/cli" -Name "prompt-manager"
