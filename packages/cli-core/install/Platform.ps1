function Get-VrooliDefaultInstallDir {
	$homeDir = $env:HOME
	if (-not $homeDir) { $homeDir = $env:USERPROFILE }
	if (-not $homeDir) { $homeDir = [Environment]::GetFolderPath('UserProfile') }
	if (-not $homeDir) { throw 'Unable to resolve home directory for the Vrooli install path.' }
	return (Join-Path $homeDir '.vrooli/bin')
}

function Get-VrooliArchitecture {
	$arch = $env:PROCESSOR_ARCHITECTURE
	if ([Environment]::Is64BitOperatingSystem -and $env:PROCESSOR_ARCHITEW6432) {
		$arch = $env:PROCESSOR_ARCHITEW6432
	}
	switch -Regex ($arch) {
		'^(AMD64|x86_64)$' { return 'amd64' }
		'^(ARM64|aarch64)$' { return 'arm64' }
		default { throw "Unsupported Windows architecture: $arch" }
	}
}

function Invoke-VrooliDownload([string]$Uri, [string]$OutFile) {
	Invoke-WebRequest -UseBasicParsing -Uri $Uri -OutFile $OutFile
}

function Get-VrooliExpectedChecksum([string]$Manifest, [string]$Asset) {
	foreach ($line in Get-Content -LiteralPath $Manifest) {
		$parts = $line -split '\s+', 2
		if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $Asset) { return $parts[0] }
	}
	throw "Release checksum manifest does not contain $Asset"
}

function Test-VrooliChecksum([string]$Manifest, [string]$Asset, [string]$Path) {
	$expected = Get-VrooliExpectedChecksum $Manifest $Asset
	$actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
	if ($actual -ne $expected.ToLowerInvariant()) {
		throw "Checksum mismatch for $Asset (expected $expected, got $actual)"
	}
}

function Test-VrooliManifestSignature([string]$Manifest, [string]$SignatureBase64) {
	# RSA public parameters for install/vrooli-release.pub. RSAParameters works
	# on Windows PowerShell 5.1 as well as modern PowerShell, without OpenSSL.
	$modulus = 'k/E+Lt7yJyj29CnISiMQku6yRtOVWogzQAvdXE/st5MelBng4yrZ3zaSef90oS/TU2pkaWV/TAnc4P1EAdyP7UCH1pVjA1dBFoN4XAGpUsRoH+BSMmhApXfc9UayZPxtxr+HCF16csbMC7k++RRSk6/s7g35qdnthYpDMhRuIrs+46kJs9eoYg7nBXRyFaBM2u2ObP4KKZQLVV1vQ+OfWaGjQW2FU2iID2vtdHc9qC/HYOS8ZA+N1UCODaaGKtA5qoJCGtBsq/+VG+j4ETcmivlYwS2xf4VXCzgpbaKmENUC36XQDjfhs1L8zopDP009z49xAMifXjOdJ31aogMPt01qRUnYVmNdZu7QWvLax9LWAIom6NfGEBx5IHA95Zt2c10hqBv5TKuyvTLbUi/DJsb5xgwGst/jtxHCKdidaBYIJMjgD/d/Z5dpkH/rdC0XPWjEcfXc5QW8UkSD8g66e14ffzXNVhG50qxNV8UaiKNcFO/fqm7mLag5qY7udEVV'
	$rsa = New-Object System.Security.Cryptography.RSACryptoServiceProvider
	try {
		$params = New-Object System.Security.Cryptography.RSAParameters
		$params.Modulus = [Convert]::FromBase64String($modulus)
		$params.Exponent = [byte[]](1, 0, 1)
		$rsa.ImportParameters($params)
		$data = [IO.File]::ReadAllBytes($Manifest)
		$signature = [Convert]::FromBase64String((Get-Content -Raw -LiteralPath $SignatureBase64).Trim())
		if (-not $rsa.VerifyData($data, 'SHA256', $signature)) { throw 'Release signature verification failed.' }
	}
	finally { $rsa.Dispose() }
}
