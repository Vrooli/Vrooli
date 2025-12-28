// SSH Key Types
export type SSHKeyType = "ed25519" | "rsa" | "ecdsa" | "dsa" | "unknown";

export interface SSHKeyInfo {
  path: string;
  type: SSHKeyType;
  bits?: number;
  fingerprint: string;
  comment?: string;
  created_at?: string;
}

export type SSHConnectionStatus =
  | "untested"
  | "testing"
  | "connected"
  | "auth_failed"
  | "host_unreachable"
  | "timeout"
  | "key_not_found"
  | "permission_denied"
  | "unknown_error";

// API Response Types
export interface ListSSHKeysResponse {
  keys: SSHKeyInfo[];
  ssh_dir: string;
  timestamp: string;
}

export interface GenerateSSHKeyRequest {
  type: SSHKeyType;
  bits?: number;
  comment?: string;
  filename?: string;
  password?: string;
}

export interface GenerateSSHKeyResponse {
  key: SSHKeyInfo;
  timestamp: string;
}

export interface GetPublicKeyRequest {
  key_path: string;
}

export interface GetPublicKeyResponse {
  public_key: string;
  fingerprint: string;
  timestamp: string;
}

export interface TestSSHConnectionRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
}

export interface TestSSHConnectionResponse {
  ok: boolean;
  status: SSHConnectionStatus;
  message?: string;
  hint?: string;
  server_info?: string;
  fingerprint?: string;
  latency_ms?: number;
  timestamp: string;
}

export interface CopySSHKeyRequest {
  host: string;
  port?: number;
  user?: string;
  key_path: string;
  password: string;
}

export type CopySSHKeyStatus =
  | "copied"
  | "already_exists"
  | "auth_failed"
  | "error";

export interface CopySSHKeyResponse {
  ok: boolean;
  status: CopySSHKeyStatus;
  message?: string;
  hint?: string;
  key_copied: boolean;
  already_exists: boolean;
  timestamp: string;
}

// Error hints for UI display
export const SSH_ERROR_HINTS: Record<
  SSHConnectionStatus,
  { title: string; hints: string[] }
> = {
  untested: {
    title: "Not Tested",
    hints: ["Click 'Test Connection' to verify SSH access"],
  },
  testing: {
    title: "Testing Connection",
    hints: ["Please wait while we verify SSH access..."],
  },
  connected: {
    title: "Connected",
    hints: ["SSH connection is working"],
  },
  auth_failed: {
    title: "Authentication Failed",
    hints: [
      "The SSH key was not accepted by the server",
      "Use 'Copy Key to Server' to add your key",
      "Or check that the correct key is selected",
    ],
  },
  host_unreachable: {
    title: "Host Unreachable",
    hints: [
      "Verify the IP address or hostname is correct",
      "Check that SSH port (usually 22) is open",
      "Ensure the VPS is powered on and running",
      "Check for firewall rules blocking SSH",
    ],
  },
  timeout: {
    title: "Connection Timed Out",
    hints: [
      "The server did not respond within 10 seconds",
      "This may indicate network issues or firewall blocking",
      "Try again or check server logs",
    ],
  },
  key_not_found: {
    title: "Key Not Found",
    hints: [
      "The selected key file could not be found",
      "It may have been deleted or moved",
      "Select a different key or generate a new one",
    ],
  },
  permission_denied: {
    title: "Permission Denied",
    hints: [
      "The SSH key file may have incorrect permissions",
      "Run: chmod 600 ~/.ssh/your_key",
      "Ensure the key file is owned by your user",
    ],
  },
  unknown_error: {
    title: "Unknown Error",
    hints: [
      "An unexpected error occurred",
      "Check the browser console for details",
      "Try refreshing the page",
    ],
  },
};
