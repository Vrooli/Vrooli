import "@testing-library/jest-dom";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

// provider-free-exception: SSH key setup is controlled by mocked API hooks and does not consume a provider.

const api = vi.hoisted(() => ({
  listSSHKeys: vi.fn(),
  generateSSHKey: vi.fn(),
  testSSHConnection: vi.fn(),
  copySSHKey: vi.fn(),
  getPublicKey: vi.fn(),
  deleteSSHKey: vi.fn(),
}));

vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
  return { ...actual, ...api };
});

import { SSHKeySetup } from "./SSHKeySetup";
import type { SSHConnectionStatus, SSHKeyInfo } from "../../types/ssh";

const key: SSHKeyInfo = {
  path: "/home/test/.ssh/deploy",
  type: "ed25519",
  fingerprint: "SHA256:fingerprint",
};

function renderSetup(
  overrides: Partial<React.ComponentProps<typeof SSHKeySetup>> = {},
) {
  const props: React.ComponentProps<typeof SSHKeySetup> = {
    host: "vps.example.test",
    port: 2222,
    user: "deploy",
    selectedKeyPath: null,
    onKeyPathChange: vi.fn(),
    onConnectionStatusChange: vi.fn(),
    ...overrides,
  };
  return { ...render(<SSHKeySetup {...props} />), props };
}

describe("SSHKeySetup", () => {
  let clipboardWriteText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    api.listSSHKeys.mockResolvedValue({ keys: [key], ssh_dir: "/home/test/.ssh", timestamp: "now" });
    api.generateSSHKey.mockResolvedValue({ key: { ...key, path: "/home/test/.ssh/generated" }, timestamp: "now" });
    api.testSSHConnection.mockResolvedValue({
      ok: true,
      status: "success",
      message: "Connected",
      server_info: "Ubuntu 24.04",
      timestamp: "now",
    });
    api.copySSHKey.mockResolvedValue({
      ok: true,
      status: "success",
      key_copied: true,
      already_exists: false,
      message: "Copied",
      timestamp: "now",
    });
    api.getPublicKey.mockResolvedValue({ public_key: "ssh-ed25519 AAAA", fingerprint: key.fingerprint, timestamp: "now" });
    api.deleteSSHKey.mockResolvedValue({ ok: true, private_deleted: true, public_deleted: true, timestamp: "now" });
    clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: clipboardWriteText } });
  });

  it("loads keys, selects one, and reports successful connection details", async () => {
    const onKeyPathChange = vi.fn();
    const onConnectionStatusChange = vi.fn();
    renderSetup({ selectedKeyPath: key.path, onKeyPathChange, onConnectionStatusChange });

    await waitFor(() => expect(screen.getByRole("button", { name: /deploy/ })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /deploy/ }));
    fireEvent.click(screen.getByText("deploy", { exact: true }));
    expect(onKeyPathChange).toHaveBeenCalledWith(key.path);
    fireEvent.click(screen.getByRole("button", { name: "Test Connection" }));

    await waitFor(() => expect(api.testSSHConnection).toHaveBeenCalledWith({
      host: "vps.example.test",
      port: 2222,
      user: "deploy",
      key_path: key.path,
    }));
    expect(await screen.findByText("Connected successfully")).toBeInTheDocument();
    expect(screen.getByText("Ubuntu 24.04")).toBeInTheDocument();
    expect(onConnectionStatusChange).toHaveBeenCalledWith("success");
  });

  it("surfaces authentication recovery, public-key copy, and copy failures", async () => {
    api.testSSHConnection.mockResolvedValue({ ok: false, status: "auth_failed", message: "Denied", timestamp: "now" });
    api.copySSHKey.mockRejectedValue(new Error("password rejected"));
    const selected = key.path;
    renderSetup({ selectedKeyPath: selected });

    await waitFor(() => expect(screen.getByRole("button", { name: "Test Connection" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Test Connection" }));
    expect(await screen.findByText("Copy SSH Key to Server")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Enter SSH password"), { target: { value: "secret" } });
    fireEvent.click(screen.getByRole("button", { name: "Copy Key to Server" }));
    expect(await screen.findByText("password rejected")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Show public key/ }));
    expect(await screen.findByText("ssh-ed25519 AAAA")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^Copy$/ }));
    await waitFor(() => expect(clipboardWriteText).toHaveBeenCalledWith("ssh-ed25519 AAAA"));
  });

  it("generates a key and handles delete confirmation failures and success", async () => {
    const onKeyPathChange = vi.fn();
    renderSetup({ onKeyPathChange });
    await waitFor(() => expect(screen.getByRole("button", { name: "Select a key..." })).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: "Generate new key" }));
    fireEvent.change(screen.getByLabelText("Key Name"), { target: { value: "new-key" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate Key" }));
    await waitFor(() => expect(api.generateSSHKey).toHaveBeenCalledWith({
      type: "ed25519",
      filename: "new-key",
      password: undefined,
      comment: "Generated by Vrooli for VPS deployment",
    }));
    expect(onKeyPathChange).toHaveBeenCalledWith("/home/test/.ssh/generated");

    fireEvent.click(screen.getByRole("button", { name: "Select a key..." }));
    const deleteButtons = screen.getAllByTitle("Delete key");
    const firstDeleteButton = deleteButtons[0];
    if (!firstDeleteButton) throw new Error("expected a delete button for the selected key");
    fireEvent.click(firstDeleteButton);
    expect(screen.getByText("Delete SSH Key?")).toBeInTheDocument();
    api.deleteSSHKey.mockResolvedValueOnce({ ok: false, message: "still in use", private_deleted: false, public_deleted: false, timestamp: "now" });
    fireEvent.click(screen.getByRole("button", { name: "Delete Key" }));
    expect(await screen.findByText("still in use")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete Key" }));
    await waitFor(() => expect(api.deleteSSHKey).toHaveBeenCalledTimes(2));
  });

  it("renders key discovery errors and normalizes unknown connection statuses", async () => {
    api.listSSHKeys.mockRejectedValue(new Error("ssh directory unavailable"));
    api.testSSHConnection.mockResolvedValue({ ok: false, status: "new_status", message: "unknown", timestamp: "now" });
    const onConnectionStatusChange = vi.fn<(status: SSHConnectionStatus) => void>();
    renderSetup({ selectedKeyPath: key.path, onConnectionStatusChange });
    fireEvent.click(screen.getByRole("button", { name: "Test Connection" }));
    await waitFor(() => expect(screen.getByText("unknown")).toBeInTheDocument());
    expect(onConnectionStatusChange).toHaveBeenCalledWith("unknown_error");
    fireEvent.click(screen.getByRole("button", { name: "No keys found" }));
    expect(await screen.findByText("ssh directory unavailable")).toBeInTheDocument();
  });

  it("renders every supported connection failure state with its recovery guidance", async () => {
    const statuses = [
      "host_unreachable", "timeout", "not_found", "ipv6_unavailable", "host_key_changed",
      "key_error", "dns_failed", "disk_full", "error",
    ] as const;
    for (const status of statuses) {
      cleanup();
      api.testSSHConnection.mockResolvedValueOnce({ ok: false, status, message: `${status} message`, hint: `${status} hint`, timestamp: "now" });
      renderSetup({ selectedKeyPath: key.path });
      const button = await screen.findByRole("button", { name: "Test Connection" });
      fireEvent.click(button);
      expect(await screen.findByText(`${status} message`)).toBeInTheDocument();
    }
  });

  it("supports RSA passphrases and reports key/public-key failures", async () => {
    api.generateSSHKey.mockRejectedValueOnce(new Error("key generation unavailable"));
    api.getPublicKey.mockRejectedValueOnce(new Error("public key unavailable"));
    api.testSSHConnection.mockResolvedValueOnce({ ok: false, status: "auth_failed", message: "Denied", timestamp: "now" });
    renderSetup({ selectedKeyPath: key.path });
    await waitFor(() => expect(screen.getByRole("button", { name: "Generate new key" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Generate new key" }));
    fireEvent.change(screen.getByLabelText("Key Name"), { target: { value: "rsa-key" } });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "rsa" } });
    fireEvent.change(screen.getByPlaceholderText("Leave empty for no passphrase"), { target: { value: "secret" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate Key" }));
    expect(await screen.findByText("key generation unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Test Connection" }));
    expect(await screen.findByText("Copy SSH Key to Server")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Show public key/ }));
    await waitFor(() => expect(api.getPublicKey).toHaveBeenCalled());
  });
});
