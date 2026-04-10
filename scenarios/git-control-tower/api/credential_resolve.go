package main

import (
	"context"
	"log"

	"git-control-tower/ssh"
)

// resolveCredentialForRemote returns the best available credential for a git
// remote. It first checks the credential store for an explicitly saved
// credential. If none is found and the remote URL uses the SSH protocol, it
// discovers SSH keys in ~/.ssh and returns a synthetic credential that sets
// GIT_SSH_COMMAND, making git operations independent of the SSH agent.
//
// This fallback prevents authentication failures after system restarts when no
// credential has been explicitly saved but SSH keys exist on disk.
func resolveCredentialForRemote(ctx context.Context, git GitRunner, credStore *CredentialsStore, repoDir, remote string) *StoredCredential {
	if remote == "" {
		remote = "origin"
	}

	// Try stored credential first.
	if credStore != nil {
		cred, _ := credStore.GetCredentialByRemote(remote)
		if cred != nil {
			return cred
		}
	}

	// Fallback: if the remote URL uses SSH and SSH keys exist on disk,
	// construct a synthetic credential so that gitCredentialEnv sets
	// GIT_SSH_COMMAND explicitly (no SSH agent dependency).
	if git == nil {
		return nil
	}

	remoteURL, err := git.GetRemoteURL(ctx, repoDir, remote)
	if err != nil {
		return nil
	}

	if detectCredentialType(remoteURL) != CredentialTypeSSH {
		return nil
	}

	platform := ssh.DefaultPlatform()
	keys, err := ssh.DiscoverKeys(platform, "")
	if err != nil || len(keys) == 0 {
		return nil
	}

	log.Printf("INFO: no stored credential for remote %q, using discovered SSH key %s", remote, keys[0].Path)
	return &StoredCredential{
		Type:       CredentialTypeSSH,
		SSHKeyPath: keys[0].Path,
	}
}
