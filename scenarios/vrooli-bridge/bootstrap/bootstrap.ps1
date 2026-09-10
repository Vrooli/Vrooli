param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$BootstrapArgs
)

# Native Windows counterpart to bootstrap.sh. Progress markers stay on stdout;
# diagnostics stay on stderr so the onboarding transport can persist the same
# durable step envelope on every platform. The pairing code is read from the
# first stdin line and never appears in argv.
$ErrorActionPreference = 'Stop'
$MarkerVersion = 1
$CurrentStep = ''

function Write-Log([string]$Message) {
    [Console]::Error.WriteLine($Message)
}

function Write-Marker([string]$Event, [string]$Step = '', [string]$Detail = '') {
    $line = "VBOOTSTRAP v=$MarkerVersion event=$Event"
    if ($Step) { $line += " step=$Step" }
    if ($Detail) {
        $safe = $Detail.Replace('"', "'").Replace("`r", ' ').Replace("`n", ' ')
        $line += (' detail="' + $safe + '"')
    }
    Write-Output $line
}

function Start-Step([string]$Step, [string]$Detail) {
    $script:CurrentStep = $Step
    Write-Marker 'step-start' $Step $Detail
    Write-Log "==> ${Step}: ${Detail}"
}

function Complete-Step([string]$Detail = '') {
    Write-Marker 'step-ok' $script:CurrentStep $Detail
    Write-Log "    ok: $($script:CurrentStep) $Detail"
    $script:CurrentStep = ''
}

function Skip-Step([string]$Detail) {
    Write-Marker 'step-skip' $script:CurrentStep $Detail
    Write-Log "    skip: $($script:CurrentStep) $Detail"
    $script:CurrentStep = ''
}

function Stop-Bootstrap([int]$Code, [string]$Detail) {
    if ($script:CurrentStep) {
        Write-Marker 'step-fail' $script:CurrentStep $Detail
        Write-Log "    FAIL: $($script:CurrentStep): $Detail"
    }
    Write-Marker 'run-fail' '' $Detail
    exit $Code
}

trap {
    $detail = $_.Exception.Message
    if (-not $detail) { $detail = 'unexpected PowerShell bootstrap error' }
    Stop-Bootstrap 1 $detail
}

function Get-Option([hashtable]$Options, [string]$Name, [string]$EnvironmentName, [string]$Default = '') {
    if ($Options.ContainsKey($Name)) { return [string]$Options[$Name] }
    $envValue = [Environment]::GetEnvironmentVariable($EnvironmentName)
    if ($null -ne $envValue) { return [string]$envValue }
    return $Default
}

function Get-Flag([hashtable]$Flags, [string]$Name) {
    return $Flags.ContainsKey($Name) -and [bool]$Flags[$Name]
}

function Parse-Arguments([string[]]$Values) {
    $options = @{}
    $flags = @{}
    for ($i = 0; $i -lt $Values.Count; $i++) {
        $name = $Values[$i]
        if ($name -eq '-h' -or $name -eq '--help') {
            Write-Log 'Usage: bootstrap.ps1 --control-plane-url URL [options]'
            exit 0
        }
        switch ($name) {
            '--include-optional' { $flags[$name] = $true; continue }
            '--credential-passphrase-stdin' { $flags[$name] = $true; continue }
            '--reconcile-pairing' { $flags[$name] = $true; continue }
            '--skip-prereqs' { $flags[$name] = $true; continue }
            '--skip-setup' { $flags[$name] = $true; continue }
            '--force-setup' { $flags[$name] = $true; continue }
            default {
                if ($i + 1 -ge $Values.Count) { Stop-Bootstrap 2 "missing value for $name" }
                $options[$name] = $Values[$i + 1]
                $i++
            }
        }
    }
    return @($options, $flags)
}

$parsed = Parse-Arguments $BootstrapArgs
$Options = $parsed[0]
$Flags = $parsed[1]

$ControlPlaneURL = Get-Option $Options '--control-plane-url' 'BRIDGE_CONTROL_PLANE_URL'
if (-not $ControlPlaneURL) { Stop-Bootstrap 2 'control-plane URL is required' }
$NodeName = Get-Option $Options '--node-name' 'BRIDGE_NODE_NAME' $env:COMPUTERNAME
$RepoURL = Get-Option $Options '--repo-url' 'BRIDGE_REPO_URL' 'https://github.com/Vrooli/Vrooli.git'
$Revision = Get-Option $Options '--revision' 'BRIDGE_REVISION'
$CheckoutDir = Get-Option $Options '--checkout-dir' 'BRIDGE_CHECKOUT_DIR' (Join-Path $HOME 'Vrooli')
$SourceDir = Get-Option $Options '--source-dir' 'BRIDGE_SOURCE_DIR'
$SourceDigest = Get-Option $Options '--source-digest' 'BRIDGE_SOURCE_DIGEST'
$LocalAppData = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { Join-Path $HOME 'AppData\Local' }
$StateDir = Get-Option $Options '--state-dir' 'BRIDGE_AGENT_STATE_DIR' (Join-Path $LocalAppData 'Vrooli\Bridge\agent')
$WorkDir = Get-Option $Options '--work-dir' 'BRIDGE_WORK_DIR'
$ServiceUser = Get-Option $Options '--service-user' 'BRIDGE_SERVICE_USER' ([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)
$ProvisionServiceUser = Get-Option $Options '--provision-service-user' 'BRIDGE_PROVISION_SERVICE_USER'
$ProvisionSocket = Get-Option $Options '--provision-socket' 'BRIDGE_PROVISION_SOCKET' '\\.\pipe\vrooli-bridge-provision'
$Capabilities = Get-Option $Options '--capabilities' 'BRIDGE_CAPABILITIES'
$PresenceOnly = Get-Option $Options '--presence-only' 'BRIDGE_PRESENCE_ONLY' 'true'
$VerifyTimeout = [int](Get-Option $Options '--verify-timeout' 'BRIDGE_VERIFY_TIMEOUT' '120')
$SetupEnvironment = Get-Option $Options '--setup-environment' 'BRIDGE_SETUP_ENVIRONMENT'
$SetupResources = Get-Option $Options '--setup-resources' 'BRIDGE_SETUP_RESOURCES'
$SetupScenarios = Get-Option $Options '--setup-scenarios' 'BRIDGE_SETUP_SCENARIOS'
$VrooliBin = Get-Option $Options '--vrooli-bin' 'BRIDGE_VROOLI_BIN'
$AgentBin = Get-Option $Options '--agent-bin' 'BRIDGE_AGENT_BIN'
$BridgeCLI = Get-Option $Options '--bridge-cli' 'BRIDGE_CLI_BIN'
$SkipPrereqs = Get-Flag $Flags '--skip-prereqs'
$SkipSetup = Get-Flag $Flags '--skip-setup'
$ForceSetup = Get-Flag $Flags '--force-setup'
$ReconcilePairing = Get-Flag $Flags '--reconcile-pairing'
$IncludeOptional = Get-Flag $Flags '--include-optional'

if (-not $WorkDir) { $WorkDir = $CheckoutDir }
$BootstrapStateDir = Join-Path $StateDir '.bootstrap'
$NodeIDFile = Join-Path $StateDir 'node_id'
$PinFile = Join-Path $StateDir 'control_plane.pub'
$InstallRecord = Join-Path $StateDir 'install-record.json'
$NodeID = ''
$RevisionSHA = ''
$SetupDigestKey = $SourceDigest
$Arch = ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture).ToString().ToLowerInvariant()
if ($Arch -eq 'x64') { $Arch = 'amd64' }
if ($Arch -ne 'amd64' -and $Arch -ne 'arm64') { Stop-Bootstrap 3 "unsupported Windows architecture $Arch" }

function Test-Command([string]$Name) {
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Test-SafeProfile([string]$Name, [string]$Value, [string]$Pattern = '^[A-Za-z0-9,_-]+$') {
    if (-not $Value) { return }
    if ($Value -notmatch $Pattern) { Stop-Bootstrap 2 "--$Name contains unsupported characters" }
}

Test-SafeProfile 'setup-environment' $SetupEnvironment '^(development|production|minimal)$'
Test-SafeProfile 'setup-resources' $SetupResources
Test-SafeProfile 'setup-scenarios' $SetupScenarios

function Invoke-Captured([string]$File, [object[]]$Arguments) {
    $output = (& $File @Arguments 2>&1 | Out-String)
    $code = if ($null -eq $LASTEXITCODE) { 0 } else { [int]$LASTEXITCODE }
    return @{ Output = $output; ExitCode = $code }
}

function Get-ShortHash([string]$Value) {
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        $bytes = [Text.Encoding]::UTF8.GetBytes($Value)
        return ([BitConverter]::ToString($sha.ComputeHash($bytes))).Replace('-', '').Substring(0, 16).ToLowerInvariant()
    }
    finally { $sha.Dispose() }
}

function Get-SetupArguments {
    $result = @()
    if ($SetupEnvironment) { $result += @('--environment', $SetupEnvironment) }
    if ($SetupResources) { $result += @('--resources', $SetupResources) }
    if ($SetupScenarios) { $result += @('--scenarios', $SetupScenarios) }
    if ($IncludeOptional) { $result += '--include-optional' }
    return $result
}

function Invoke-StepCommand([string]$File, [object[]]$Arguments, [string]$Failure) {
    $result = Invoke-Captured $File $Arguments
    if ($result.ExitCode -ne 0) {
        if ($result.Output) { Write-Log $result.Output.TrimEnd() }
        Stop-Bootstrap 1 "$Failure (exit $($result.ExitCode))"
    }
    return $result.Output
}

function Step-Detect {
    Start-Step 'detect-os' 'identify platform'
    Complete-Step "os=windows arch=$Arch"
}

function Step-ReceiveArtifacts {
    Start-Step 'prebuilt-artifacts' 'verify transferred prebuilt binaries'
    if (-not ($VrooliBin -and $AgentBin -and $BridgeCLI)) {
        Skip-Step 'complete prebuilt bundle not supplied; source-build fallback remains available'
        return
    }
    $fingerprint = ''
    foreach ($binary in @($VrooliBin, $AgentBin, $BridgeCLI)) {
        if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) { Stop-Bootstrap 1 "received prebuilt binary is missing: $binary" }
        $sidecar = "$binary.fp"
        if (-not (Test-Path -LiteralPath $sidecar -PathType Leaf)) { Stop-Bootstrap 1 "received binary freshness sidecar is missing: $sidecar" }
        $value = (Get-Content -Raw -LiteralPath $sidecar).Trim()
        if (-not $value) { Stop-Bootstrap 1 "received binary freshness sidecar is empty: $sidecar" }
        if (-not $fingerprint) { $fingerprint = $value }
        elseif ($fingerprint -ne $value) { Stop-Bootstrap 1 'received prebuilt binaries do not share one source fingerprint' }
    }
    Complete-Step "received prebuilt binaries for windows/$Arch (fingerprint $($fingerprint.Substring(0, [Math]::Min(12, $fingerprint.Length))))"
}

function Step-Prerequisites {
    Start-Step 'prereqs' 'ensure Windows source prerequisites'
    if ($VrooliBin -and $AgentBin -and $BridgeCLI -and $SourceDir) {
        Skip-Step 'pre-synced tree + prebuilt binaries require no clone prerequisites'
        return
    }
    if ($SkipPrereqs) { Skip-Step '--skip-prereqs'; return }
    if (-not (Test-Command 'git')) { Stop-Bootstrap 1 'git is required on Windows for pinned source onboarding' }
    Skip-Step 'git is available'
}

function Step-Clone {
    Start-Step 'clone' "clone/converge $RepoURL"
    if ($SourceDir) {
        if (-not (Test-Path -LiteralPath $SourceDir -PathType Container)) { Stop-Bootstrap 1 "--source-dir $SourceDir does not exist" }
        if (-not (Test-Path -LiteralPath (Join-Path $SourceDir '.vrooli\repo-contract.json') -PathType Leaf)) { Stop-Bootstrap 1 'pre-synced source is missing .vrooli/repo-contract.json' }
        $script:CheckoutDir = $SourceDir
        $script:RevisionSHA = if ($Revision) { "$Revision+dirty" } else { 'working-tree' }
        Complete-Step "using pre-synced working tree at $SourceDir ($RevisionSHA)"
        return
    }
    if (Test-Path -LiteralPath (Join-Path $CheckoutDir '.git')) {
        Invoke-StepCommand 'git' @('-C', $CheckoutDir, 'fetch', '--tags', '--prune', 'origin') 'git fetch failed' | Out-Null
        if ($Revision) { Invoke-StepCommand 'git' @('-C', $CheckoutDir, 'checkout', '--quiet', $Revision) 'git checkout failed' | Out-Null }
    }
    else {
        if (Test-Path -LiteralPath $CheckoutDir) {
            $existing = Get-ChildItem -Force -LiteralPath $CheckoutDir
            if ($existing.Count -gt 0) { Stop-Bootstrap 1 "checkout directory $CheckoutDir is not empty" }
        }
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $CheckoutDir) | Out-Null
        Invoke-StepCommand 'git' @('clone', $RepoURL, $CheckoutDir) 'git clone failed' | Out-Null
        if ($Revision) { Invoke-StepCommand 'git' @('-C', $CheckoutDir, 'checkout', '--quiet', $Revision) 'git checkout failed' | Out-Null }
    }
    $script:RevisionSHA = (Invoke-StepCommand 'git' @('-C', $CheckoutDir, 'rev-parse', 'HEAD') 'could not resolve checkout revision').Trim()
    Complete-Step "at $RevisionSHA"
}

function Step-PrepareVrooli {
    if ($VrooliBin) { return }
    Start-Step 'build-vrooli' 'prepare Vrooli CLI'
    if (-not (Test-Command 'go')) { Stop-Bootstrap 1 'go is required for the Windows source-build fallback' }
    $binDir = Join-Path $StateDir 'bin'
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    $script:VrooliBin = Join-Path $binDir 'vrooli.exe'
    Push-Location $CheckoutDir
    try { Invoke-StepCommand 'go' @('run', './cmd/vrooli-dist', '--root', $CheckoutDir, '--goos', 'windows', '--goarch', $Arch, '--output', $VrooliBin) 'Vrooli CLI build failed' | Out-Null }
    finally { Pop-Location }
    if (-not (Test-Path -LiteralPath $VrooliBin -PathType Leaf)) { Stop-Bootstrap 1 'Vrooli CLI build produced no executable' }
    Complete-Step "prepared $VrooliBin"
}

function Step-Setup {
    Start-Step 'setup' 'vrooli setup'
    if ($SkipSetup) { Skip-Step '--skip-setup (node cannot run jobs until setup is run later)'; return }
    New-Item -ItemType Directory -Force -Path $BootstrapStateDir | Out-Null
    $profile = ((@($RevisionSHA, $SetupEnvironment, $SetupResources, $SetupScenarios, $IncludeOptional, $SetupDigestKey) -join '|'))
    $sentinel = Join-Path $BootstrapStateDir ("setup-" + (Get-ShortHash $profile) + '.done')
    if (-not $ForceSetup -and (Test-Path -LiteralPath $sentinel)) { Skip-Step "already set up at $RevisionSHA for this profile"; return }
    $resultFile = Join-Path $BootstrapStateDir 'setup-result.json'
    $oldSource = $env:VROOLI_SOURCE_ROOT
    $env:VROOLI_SOURCE_ROOT = $CheckoutDir
    try {
        $setupArgs = @('setup') + (Get-SetupArguments) + @('--result-file', $resultFile)
        Invoke-StepCommand $VrooliBin $setupArgs 'vrooli setup failed' | Out-Null
    }
    finally {
        if ($null -eq $oldSource) { Remove-Item Env:VROOLI_SOURCE_ROOT -ErrorAction SilentlyContinue }
        else { $env:VROOLI_SOURCE_ROOT = $oldSource }
    }
    New-Item -ItemType File -Force -Path $sentinel | Out-Null
    Complete-Step "setup complete at $RevisionSHA"
}

function Step-BuildAgent {
    Start-Step 'build-agent' 'prepare node-agent'
    if ($AgentBin) { Skip-Step "received prebuilt $AgentBin; no node-side build"; return }
    if (-not (Test-Command 'go')) { Stop-Bootstrap 1 'go is required to build the Windows node-agent' }
    $binDir = Join-Path $StateDir 'bin'
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    $script:AgentBin = Join-Path $binDir 'vrooli-bridge-agent.exe'
    Push-Location $CheckoutDir
    try { Invoke-StepCommand 'go' @('build', '-trimpath', '-o', $AgentBin, './scenarios/vrooli-bridge/agent') 'node-agent build failed' | Out-Null }
    finally { Pop-Location }
    Complete-Step "built $AgentBin"
}

function Step-BuildCLI {
    Start-Step 'build-cli' 'prepare vrooli-bridge CLI'
    if ($BridgeCLI) { Skip-Step "received prebuilt $BridgeCLI; no node-side build"; return }
    if (-not (Test-Command 'go')) { Stop-Bootstrap 1 'go is required to build the Windows Bridge CLI' }
    $binDir = Join-Path $StateDir 'bin'
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    $script:BridgeCLI = Join-Path $binDir 'vrooli-bridge.exe'
    Push-Location $CheckoutDir
    try { Invoke-StepCommand 'go' @('build', '-trimpath', '-o', $BridgeCLI, './scenarios/vrooli-bridge/cli') 'Bridge CLI build failed' | Out-Null }
    finally { Pop-Location }
    Complete-Step "built $BridgeCLI"
}

function Step-NodeKey {
    Start-Step 'node-key' 'generate/load node keypair'
    New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
    $script:NodePublicKey = (& $AgentBin '--print-public-key' '--state-dir' $StateDir | Out-String).Trim()
    if (-not $NodePublicKey) { Stop-Bootstrap 1 'agent produced no public key' }
    Complete-Step "node public key ready (fingerprint $(Get-ShortHash $NodePublicKey))"
}

function Step-PairRedeem {
    Start-Step 'pair-redeem' 'redeem pairing code + pin control-plane key'
    if (-not $ReconcilePairing -and (Test-Path -LiteralPath $PinFile) -and (Test-Path -LiteralPath $NodeIDFile)) {
        $script:NodeID = (Get-Content -Raw -LiteralPath $NodeIDFile).Trim()
        Skip-Step "already paired as $NodeID"
        return
    }
    if (-not $env:BRIDGE_PAIRING_CODE) { Stop-Bootstrap 2 'not yet paired and BRIDGE_PAIRING_CODE is unset' }
    $oldCode = $env:BRIDGE_PAIRING_CODE
    try {
        $args = @('--api-base', $ControlPlaneURL, 'pair', 'redeem', '--public-key', $NodePublicKey, '--name', $NodeName, '--os', 'windows', '--arch', $Arch, '--state-dir', $StateDir, '--json')
        if ($Capabilities) { $args += @('--capabilities', $Capabilities) }
        $output = Invoke-StepCommand $BridgeCLI $args 'pairing redeem failed'
        $result = $output | ConvertFrom-Json
        $script:NodeID = [string]$result.node_id
    }
    finally { $env:BRIDGE_PAIRING_CODE = $oldCode }
    if (-not $NodeID) { Stop-Bootstrap 1 'pairing redeem returned no node id' }
    Set-Content -NoNewline -LiteralPath $NodeIDFile -Value $NodeID
    Complete-Step "paired as $NodeID"
}

function Step-PinVerify {
    Start-Step 'pin-verify' 'verify pinned control-plane key'
    if (-not (Test-Path -LiteralPath $PinFile -PathType Leaf)) { Stop-Bootstrap 1 "pinned control-plane key missing at $PinFile" }
    if (-not $NodeID -and (Test-Path -LiteralPath $NodeIDFile)) { $script:NodeID = (Get-Content -Raw -LiteralPath $NodeIDFile).Trim() }
    if (-not $NodeID) { Stop-Bootstrap 1 'recorded node id is missing' }
    Complete-Step "pinned key present, node $NodeID"
}

function Get-ServiceArguments([switch]$Helper) {
    if ($Helper) {
        return @('--state-dir', $StateDir, '--provision-helper', '--provision-socket', $ProvisionSocket, '--provision-client-user', $ServiceUser, '--provision-client-home', $HOME, '--service-user', $ProvisionServiceUser, '--system-service', '--work-dir', $WorkDir, '--vrooli-bin', $VrooliBin)
    }
    $result = @('--control-plane-url', $ControlPlaneURL, '--node-id', $NodeID, '--state-dir', $StateDir, '--work-dir', $WorkDir, '--service-user', $ServiceUser, "--presence-only=$PresenceOnly", '--vrooli-bin', $VrooliBin)
    if ($Capabilities) { $result += @('--capabilities', $Capabilities) }
    return $result
}

function Step-Provisioner {
    Start-Step 'provisioner-install' 'install privileged provisioning helper'
    if (-not $ProvisionServiceUser) { Skip-Step 'BRIDGE_PROVISION_SERVICE_USER is unset'; return }
    $args = Get-ServiceArguments -Helper
    $output = Invoke-StepCommand $AgentBin (@('service', 'install') + $args + @('--json')) 'privileged provisioning helper install failed'
    $result = $output | ConvertFrom-Json
    if (-not $result.running) { Stop-Bootstrap 1 'provisioning helper installed but is not running' }
    Complete-Step "helper running as $ProvisionServiceUser; runner $ServiceUser; pipe $ProvisionSocket"
}

function Step-ServiceInstall {
    Start-Step 'service-install' 'install + start node-agent service'
    $args = Get-ServiceArguments
    $status = Invoke-Captured $AgentBin (@('service', 'status') + $args + @('--json'))
    if ($status.Output) {
        try {
            $existing = $status.Output | ConvertFrom-Json
            if ($existing.installed -and $existing.configured -and $existing.running) {
                Skip-Step 'service already installed, configured, and running'
                return
            }
        } catch { }
    }
    $output = Invoke-StepCommand $AgentBin (@('service', 'install') + $args + @('--json')) 'Windows service install failed'
    $result = $output | ConvertFrom-Json
    if (-not $result.running -or -not $result.configured) { Stop-Bootstrap 1 'Windows service installed but did not reach configured running state' }
    Complete-Step "service installed and running as $ServiceUser"
}

function Step-Autostart {
    Start-Step 'autostart' 'enable native Windows auto-start'
    Complete-Step 'SCM start=auto provides reboot persistence'
}

function Step-RecordInstall {
    Start-Step 'install-record' 'record bootstrap-owned artifacts'
    $entries = @(
        [pscustomobject]@{ scope = 'runtime'; kind = 'directory'; path = $CheckoutDir; prefix = $CheckoutDir },
        [pscustomobject]@{ scope = 'agent'; kind = 'directory'; path = $StateDir; prefix = $StateDir },
        [pscustomobject]@{ scope = 'agent'; kind = 'binary'; path = $AgentBin; prefix = (Split-Path -Parent $AgentBin) },
        [pscustomobject]@{ scope = 'agent'; kind = 'binary'; path = $BridgeCLI; prefix = (Split-Path -Parent $BridgeCLI) }
    )
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $InstallRecord) | Out-Null
    [pscustomobject]@{ version = 1; prefix = (Split-Path -Parent $StateDir); updated_at = (Get-Date).ToUniversalTime().ToString('o'); entries = $entries } |
        ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $InstallRecord
    Complete-Step "bootstrap-owned paths recorded at $InstallRecord"
}

function Step-VerifyOnline {
    Start-Step 'verify-online' 'confirm service and dial-out channel'
    $deadline = (Get-Date).AddSeconds($VerifyTimeout)
    $args = Get-ServiceArguments
    while ((Get-Date) -lt $deadline) {
        $status = Invoke-Captured $AgentBin (@('service', 'status') + $args + @('--json'))
        if ($status.Output) {
            try {
                $result = $status.Output | ConvertFrom-Json
                if ($result.running) {
                    $logs = @((Join-Path $StateDir 'agent.stdout.log'), (Join-Path $StateDir 'agent.stderr.log'))
                    $connected = $false
                    foreach ($log in $logs) { if (Test-Path -LiteralPath $log) { $connected = $connected -or [bool](Select-String -Quiet -LiteralPath $log -Pattern 'dial-out stream open') } }
                    if ($connected) { Complete-Step 'agent connected (dial-out stream open)' }
                    else { Complete-Step 'service running (Windows log has not exposed the channel marker yet)' }
                    return
                }
            } catch { }
        }
        Start-Sleep -Seconds 2
    }
    Stop-Bootstrap 1 "agent did not reach running state within ${VerifyTimeout}s"
}

Write-Marker 'run-start' '' 'vrooli-bridge node bootstrap'
Write-Log "vrooli-bridge Windows bootstrap: node=$NodeName cp=$ControlPlaneURL checkout=$CheckoutDir state=$StateDir"

# The remote driver writes the code as the first stdin line. Keep it in the
# environment only long enough for the Bridge CLI to redeem it.
$env:BRIDGE_PAIRING_CODE = [Console]::In.ReadLine()

Step-Detect
Step-ReceiveArtifacts
Step-Prerequisites
Step-Clone
if (-not (Test-Path -LiteralPath $StateDir)) { New-Item -ItemType Directory -Force -Path $StateDir | Out-Null }
Step-PrepareVrooli
Step-Setup
Step-BuildAgent
Step-BuildCLI
Step-NodeKey
Step-PairRedeem
Write-Output ('VBOOTSTRAP event=node-id node-id=' + $NodeID + ' detail=""')
Step-PinVerify
Step-Provisioner
Step-ServiceInstall
Step-Autostart
Step-RecordInstall
Step-VerifyOnline
Write-Marker 'run-ok' '' "node $NodeID paired and online"
Write-Log "bootstrap complete: node $NodeID paired and online"
