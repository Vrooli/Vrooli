package generation

import (
	"fmt"
	"strconv"
	"strings"

	"scenario-to-desktop-api/signing/types"
)

// generateLinuxArtifactSigner creates the electron-builder hook used for
// Linux packages. The hook signs the finished artifact, never embeds a
// passphrase, and publishes a digest-bound metadata sidecar alongside the
// detached signatures.
func generateLinuxArtifactSigner(config *types.LinuxSigningConfig) ([]byte, error) {
	if config == nil || strings.TrimSpace(config.GPGKeyID) == "" {
		return nil, fmt.Errorf("Linux artifact signing requires a GPG key ID")
	}

	const template = `'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs/promises');
const path = require('node:path');
const { spawn } = require('node:child_process');

const GPG_KEY_ID = __GPG_KEY_ID__;
const PASSPHRASE_ENV = __PASSPHRASE_ENV__;
const GPG_HOMEDIR = __GPG_HOMEDIR__;

function runGpg(args, input) {
  return new Promise((resolve, reject) => {
    const child = spawn('gpg', args, { stdio: ['pipe', 'ignore', 'pipe'] });
    let stderr = '';
    child.stderr.on('data', (chunk) => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', (code) => {
      if (code !== 0) {
        reject(new Error('gpg exited with ' + code + ': ' + stderr.trim()));
        return;
      }
      resolve();
    });
    if (input !== undefined) child.stdin.end(input + '\\n');
    else child.stdin.end();
  });
}

function architectureFor(artifactPath) {
  const match = path.basename(artifactPath).match(/(?:^|[-_.])(x64|arm64|armv7l|ia32|universal)(?:[-_.]|$)/i);
  if (!match) throw new Error('cannot determine Linux artifact architecture from ' + artifactPath);
  return match[1].toLowerCase();
}

module.exports = async function signLinuxArtifacts(result) {
  if (!result || !Array.isArray(result.artifactPaths) || !result.outDir) {
    throw new Error('electron-builder did not provide Linux artifact paths');
  }
  const artifacts = result.artifactPaths.filter((value) => /\.(AppImage|deb|rpm|snap|tar\.gz)$/i.test(value));
  if (artifacts.length === 0) return [];

  const version = result.configuration && result.configuration.extraMetadata && result.configuration.extraMetadata.version;
  const configuredChannel = result.configuration && result.configuration.publish && result.configuration.publish.channel;
  const channel = process.env.VROOLI_UPDATE_CHANNEL || configuredChannel || 'stable';
  if (!version) throw new Error('package version is required for Linux release metadata');
  const passphrase = PASSPHRASE_ENV ? process.env[PASSPHRASE_ENV] : undefined;
  if (PASSPHRASE_ENV && passphrase === undefined) throw new Error('missing GPG passphrase environment variable ' + PASSPHRASE_ENV);

  const records = [];
  for (const artifact of artifacts) {
    const signature = artifact + '.asc';
    const args = ['--batch', '--yes', '--armor', '--detach-sign', '--local-user', GPG_KEY_ID, '--output', signature];
    if (GPG_HOMEDIR) args.push('--homedir', GPG_HOMEDIR);
    if (passphrase !== undefined) args.push('--pinentry-mode', 'loopback', '--passphrase-fd', '0');
    args.push(artifact);
    await runGpg(args, passphrase);
    const [artifactBytes, signatureBytes] = await Promise.all([fs.readFile(artifact), fs.readFile(signature)]);
    records.push({
      artifact: path.basename(artifact),
      signature: path.basename(signature),
      platform: 'linux',
      architecture: architectureFor(artifact),
      sha512: crypto.createHash('sha512').update(artifactBytes).digest('hex'),
      signature_sha512: crypto.createHash('sha512').update(signatureBytes).digest('hex'),
    });
  }

  const metadataPath = path.join(result.outDir, 'linux-update-metadata.json');
  await fs.writeFile(metadataPath, JSON.stringify({ schema_version: 'vrooli.linux-update-metadata.v1', version, channel, platform: 'linux', artifacts: records }, null, 2) + '\n', { mode: 0o644 });
  return [...records.map((record) => path.join(result.outDir, record.signature)), metadataPath];
};

module.exports.default = module.exports;
`

	return []byte(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(template,
		"__GPG_KEY_ID__", strconv.Quote(config.GPGKeyID)),
		"__PASSPHRASE_ENV__", strconv.Quote(config.GPGPassphraseEnv)),
		"__GPG_HOMEDIR__", strconv.Quote(config.GPGHomedir))), nil
}
