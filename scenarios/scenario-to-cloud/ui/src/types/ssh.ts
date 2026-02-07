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
  | "success"
  | "auth_failed"
  | "host_unreachable"
  | "timeout"
  | "not_found"
  | "ipv6_unavailable"
  | "host_key_changed"
  | "key_error"
  | "dns_failed"
  | "disk_full"
  | "error"
  | "unknown_error";

export const SSH_API_OUTCOME_STATUSES = [
  "success",
  "already_exists",
  "not_found",
  "auth_failed",
  "timeout",
  "host_unreachable",
  "host_key_changed",
  "ipv6_unavailable",
  "invalid_input",
  "disk_full",
  "dns_failed",
  "key_error",
  "error",
] as const;

export const SSH_TEST_API_STATUSES = [
  "success",
  "auth_failed",
  "timeout",
  "host_unreachable",
  "host_key_changed",
  "ipv6_unavailable",
  "key_error",
  "dns_failed",
  "disk_full",
  "not_found",
  "error",
] as const;

export const SSH_COPY_KEY_API_STATUSES = [
  "success",
  "already_exists",
  "auth_failed",
  "ipv6_unavailable",
  "key_error",
  "error",
] as const;

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
  | "success"
  | "already_exists"
  | "auth_failed"
  | "ipv6_unavailable"
  | "key_error"
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

export interface DeleteSSHKeyRequest {
  key_path: string;
}

export interface DeleteSSHKeyResponse {
  ok: boolean;
  message?: string;
  private_deleted: boolean;
  public_deleted: boolean;
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
  success: {
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
  not_found: {
    title: "Key Not Found",
    hints: [
      "The selected key file could not be found",
      "It may have been deleted or moved",
      "Select a different key or generate a new one",
    ],
  },
  key_error: {
    title: "Key Error",
    hints: [
      "The SSH key file may be corrupted or have incorrect permissions",
      "Run: chmod 600 ~/.ssh/your_key",
      "Ensure the key file is owned by your user",
    ],
  },
  ipv6_unavailable: {
    title: "IPv6 Not Available",
    hints: [
      "Your network does not have IPv6 connectivity",
      "Most ISPs still only provide IPv4 addresses",
      "Use the IPv4 address of your server instead",
      "Check your VPS dashboard for the IPv4 address",
    ],
  },
  host_key_changed: {
    title: "Server Identity Changed",
    hints: [
      "The server's host key has changed since you last connected",
      "This could happen if the server was rebuilt or reinstalled",
      "If unexpected, this could indicate a security issue",
      "Run: ssh-keygen -R <host> to remove the old key",
    ],
  },
  dns_failed: {
    title: "DNS Failed",
    hints: [
      "Hostname could not be resolved",
      "Check the VPS hostname or use the server IP address",
    ],
  },
  disk_full: {
    title: "Disk Full",
    hints: [
      "The server appears to be out of disk space",
      "SSH in and run: df -h",
      "Free space and try again",
    ],
  },
  error: {
    title: "SSH Error",
    hints: [
      "SSH command failed",
      "Review the error message and server logs",
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
