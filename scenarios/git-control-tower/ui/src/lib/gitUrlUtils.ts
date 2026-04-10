// URL conversion helpers for git remote URLs

export function convertSSHToHTTPS(sshUrl: string): string {
  const trimmed = sshUrl.trim();

  // Handle git@host:user/repo.git format
  if (trimmed.startsWith("git@")) {
    const rest = trimmed.slice(4); // Remove "git@"
    const parts = rest.split(":");
    if (parts.length === 2) {
      return `https://${parts[0]}/${parts[1]}`;
    }
  }

  // Handle ssh://git@host/user/repo.git format
  if (trimmed.startsWith("ssh://")) {
    let rest = trimmed.slice(6); // Remove "ssh://"
    rest = rest.replace(/^git@/, "");
    return `https://${rest}`;
  }

  return trimmed;
}

export function convertHTTPSToSSH(httpsUrl: string): string {
  const trimmed = httpsUrl.trim();

  if (trimmed.startsWith("https://") || trimmed.startsWith("http://")) {
    const rest = trimmed.replace(/^https?:\/\//, "");
    const parts = rest.split("/");
    if (parts.length >= 2) {
      const host = parts[0];
      const path = parts.slice(1).join("/");
      return `git@${host}:${path}`;
    }
  }

  return trimmed;
}

export function detectUrlType(url: string): "https" | "ssh" {
  const trimmed = url.trim();
  if (trimmed.startsWith("git@") || trimmed.startsWith("ssh://")) {
    return "ssh";
  }
  return "https";
}
