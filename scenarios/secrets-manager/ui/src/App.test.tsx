import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { normalizeRouterBasename } from "./app/routerUtils";
import { PasswordManagerApp } from "./PasswordManagerApp";
import { ThemeProvider } from "./theme/ThemeProvider";
import { useTheme } from "./theme/ThemeContext";
import { renderWithProviders } from "./test-utils";

const api = vi.hoisted(() => ({
  status: vi.fn(),
  enrollmentStatus: vi.fn(),
  enroll: vi.fn(),
  items: vi.fn(),
  unlock: vi.fn(),
  lock: vi.fn(),
  create: vi.fn(),
  reveal: vi.fn(),
  trash: vi.fn(),
  restore: vi.fn(),
  update: vi.fn(),
  recovery: vi.fn(),
  preview: vi.fn(),
  commit: vi.fn(),
  generate: vi.fn(),
  audit: vi.fn(),
  history: vi.fn(),
  grants: vi.fn(),
  effectiveAccess: vi.fn(),
  revokeGrant: vi.fn(),
  assurance: vi.fn(),
  requests: vi.fn(),
  approve: vi.fn(),
  deny: vi.fn(),
  sources: vi.fn(),
  source: vi.fn()
  ,exportAudit: vi.fn()
  ,sourceHealth: vi.fn()
  ,login: vi.fn()
}));

vi.mock("./lib/passwordManagerApi", () => ({
  getVaultStatus: api.status,
  getEnrollmentStatus: api.enrollmentStatus,
  completeEnrollment: api.enroll,
  listVaultItems: api.items,
  unlockVault: api.unlock,
  lockVault: api.lock,
  createVaultItem: api.create,
  revealVaultItem: api.reveal,
  trashVaultItem: api.trash,
  restoreVaultItem: api.restore,
  updateVaultItem: api.update,
  getRecoveryStatus: api.recovery,
  previewVaultImport: api.preview,
  commitVaultImport: api.commit,
  generatePassword: api.generate,
  getAudit: api.audit,
  listVaultItemHistory: api.history,
  listGrants: api.grants,
  getGrantEffectiveAccess: api.effectiveAccess,
  revokeGrant: api.revokeGrant,
  issueAssurance: api.assurance,
  listAccessRequests: api.requests,
  approveAccessRequest: api.approve,
  denyAccessRequest: api.deny,
  listSources: api.sources,
  getSourceHealth: api.sourceHealth,
  exportAudit: api.exportAudit,
  createSource: api.source,
  setConfiguredOwnerToken: vi.fn(),
  loginWithAuthenticator: api.login
}));

function defaults() {
  const fixtureSecretValue = "fixture-value-" + "123";
  api.status.mockResolvedValue({ vault_id: "personal", status: "unlocked", key_available: true, supports_recovery: true });
  api.enrollmentStatus.mockResolvedValue({ workspace_id: "local", bootstrap_configured: false, enrolled: true, role_model: ["owner", "admin", "member", "viewer"] });
  api.enroll.mockResolvedValue({ owner_token: "owner-token", token_once: true, role: "owner", principal_id: "owner", workspace_id: "local" });
  api.items.mockResolvedValue({ items: [{ id: "item-1", vault_id: "personal", type: "login", name: "GitHub", username: "alice", uri: "https://github.com", tags: [], favorite: true, trashed: false, revision: 1 }] });
  api.unlock.mockResolvedValue({ status: "unlocked" });
  api.lock.mockResolvedValue({ status: "locked" });
  api.create.mockResolvedValue({});
  api.reveal.mockResolvedValue({ fields: { password: fixtureSecretValue } });
  api.trash.mockResolvedValue({});
  api.restore.mockResolvedValue({});
  api.update.mockResolvedValue({});
  api.recovery.mockResolvedValue({ key_available: true, backup_status: "not_configured", restore_status: "not_run", recovery_epoch: 1, evidence_owner: "data-backup-manager", recovery_ready: false });
  api.preview.mockResolvedValue({ version: 1, format: "native", item_count: 1, duplicate_candidates: [], unsupported_items: [], requires_duplicate_policy: false });
  api.commit.mockResolvedValue({ created: 1, skipped: 0, renamed: 0, source_vault_id: "personal" });
  api.generate.mockResolvedValue({ password: "Generated-123!" });
  api.audit.mockResolvedValue({ events: [] });
  api.history.mockResolvedValue({ history: [] });
  api.grants.mockResolvedValue({ grants: [] });
  api.requests.mockResolvedValue({ requests: [] });
  api.approve.mockResolvedValue({});
  api.deny.mockResolvedValue({});
  api.assurance.mockResolvedValue({ assurance_token: "assurance-token" });
  api.effectiveAccess.mockResolvedValue({ grant_id: "grant-1", workspace_id: "local", actor: "local-owner", item_id: "item-1", operation: "use", decision: "allow", reason: "scope permits use", selector_mode: "current_snapshot", future_members: false, raw_read_permitted: false });
  api.revokeGrant.mockResolvedValue({ grant_id: "grant-1", status: "revoked", remote_purge: "not_applicable" });
  api.sources.mockResolvedValue({ sources: [] });
  api.source.mockResolvedValue({});
}

function ThemeProbe() {
  const { choice, setTheme } = useTheme();
  return <button onClick={() => setTheme("dark")}>{choice}</button>;
}

describe("PasswordManagerApp", () => {
  afterEach(() => { cleanup(); window.history.replaceState({}, "", "/"); window.localStorage.clear(); vi.clearAllMocks(); });

  it("normalizes router bases and persists theme choices", async () => {
    expect(normalizeRouterBasename("")).toBe("/");
    expect(normalizeRouterBasename(".")).toBe("/");
    expect(normalizeRouterBasename("/secrets-manager///")).toBe("/secrets-manager");

    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() })
    });
    renderWithProviders(<ThemeProvider><ThemeProbe /></ThemeProvider>);
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("dark"));
    fireEvent.click(screen.getByRole("button", { name: "system" }));
    expect(window.localStorage.getItem("secrets-manager.theme")).toBe("dark");
    expect(document.documentElement.style.colorScheme).toBe("dark");
  });

  it("resolves stored themes and a light system preference", async () => {
    window.localStorage.setItem("secrets-manager.theme", "light");
    renderWithProviders(<ThemeProvider><ThemeProbe /></ThemeProvider>);
    expect(screen.getByRole("button", { name: "light" })).toBeInTheDocument();
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("light"));

    cleanup();
    window.localStorage.clear();
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      value: () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })
    });
    renderWithProviders(<ThemeProvider><ThemeProbe /></ThemeProvider>);
    expect(screen.getByRole("button", { name: "system" })).toBeInTheDocument();
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("light"));
  });

  it("keeps ordinary inventory metadata separate from reveal", async () => {
    defaults();
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    expect(screen.queryByText("fixture-value-123")).not.toBeInTheDocument();
    fireEvent.click(screen.getByText("GitHub"));
    fireEvent.click(screen.getByRole("button", { name: /reveal selected fields/i }));
    await waitFor(() => expect(screen.getByText(/fixture-value-123/)).toBeInTheDocument());
    expect(api.reveal).toHaveBeenCalledWith("item-1", ["password", "token", "notes"]);
  });

  it("uses the supplied page when a direct path is outside the page map", async () => {
    defaults();
    renderWithProviders(<PasswordManagerApp initialPage="settings" />, { initialEntries: ["/unknown"] });
    await waitFor(() => expect(screen.getByRole("heading", { name: "Keep the boundary visible", level: 2 })).toBeInTheDocument());
  });

  it("clears revealed values when the vault is locked", async () => {
    defaults();
    api.status.mockReset();
    api.status
      .mockResolvedValueOnce({ vault_id: "personal", status: "unlocked", key_available: true, supports_recovery: true })
      .mockResolvedValue({ vault_id: "personal", status: "locked", key_available: true, supports_recovery: true });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    fireEvent.click(screen.getByText("GitHub"));
    fireEvent.click(screen.getByRole("button", { name: /reveal selected fields/i }));
    await waitFor(() => expect(screen.getByText(/fixture-value-123/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: /lock now/i }));
    await waitFor(() => expect(screen.getByText("Vault locked")).toBeInTheDocument());
    expect(screen.queryByText(/fixture-value-123/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /unlock vault/i })).toBeInTheDocument();
  });

  it("shows the lock boundary and opens a new item form", async () => {
    defaults();
    api.status.mockResolvedValue({ vault_id: "personal", status: "locked", key_available: true, supports_recovery: true });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("Vault locked")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /unlock vault/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /new item/i }));
    expect(screen.getByText("Add to your vault")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save encrypted item/i })).toBeInTheDocument();
  });

  it("keeps protected pages visible and makes sign in user initiated", async () => {
    defaults();
    api.status.mockRejectedValue(new Error("owner authentication required"));
    api.items.mockRejectedValue(new Error("owner authentication required"));
    api.audit.mockRejectedValue(new Error("owner authentication required"));
    renderWithProviders(<App />);

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("owner authentication required"));
    expect(screen.getByText("Your vault is ready")).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: "Sign in" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getByRole("dialog", { name: "Sign in" })).toBeInTheDocument();
  });

  it("shows authentication failures inside the sign in dialog", async () => {
    defaults();
    api.status.mockRejectedValue(new Error("owner authentication required"));
    api.items.mockRejectedValue(new Error("owner authentication required"));
    api.audit.mockRejectedValue(new Error("owner authentication required"));
    api.login.mockRejectedValue(new Error("invalid credentials"));
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "owner@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "wrong-password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("invalid credentials"));

    api.login.mockRejectedValue("unstructured login failure");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Sign in failed."));
  });

  it("reports recovery evidence separately from backup readiness", async () => {
    defaults();
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Recovery" }));
    await waitFor(() => expect(screen.getByText("Authority key available")).toBeInTheDocument());
    expect(screen.getByText(/Backup: not_configured · Restore: not_run/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /preview import/i })).toBeInTheDocument();
  });

  it("surfaces tampered audit evidence in activity", async () => {
    defaults();
    api.audit.mockResolvedValue({ integrity: "tampered", events: [{ id: "event-1", actor_id: "owner", action: "item.reveal", outcome: "success", integrity_status: "tampered", created_at: "2026-09-06T00:00:00Z" }] });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Activity" }));
    expect(await screen.findByText("Evidence tampered")).toBeInTheDocument();
  });

  it("covers access, source, settings, recovery, and edit journeys", async () => {
    defaults();
    api.requests.mockResolvedValue({ requests: [{ id: "request-1", grant_id: "grant-1", item_id: "item-1", operation: "use", request_digest: "digest-1", status: "pending", requested_by: "agent-1" }] });
    api.grants.mockResolvedValue({ grants: [{ id: "grant-1", item_id: "item-1", principal_type: "agent", principal_id: "agent-1", selector_mode: "current_snapshot", operations: ["use"], status: "active", expires_at: "2026-09-07T00:00:00Z" }] });
    api.sources.mockResolvedValue({ sources: [{ id: "source-1", kind: "native", label: "Local", status: "configured", capabilities: {} }] });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: "Access" }));
    await waitFor(() => expect(screen.getByText("Access that explains itself")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Principal ID"), { target: { value: "agent-2" } });
    fireEvent.change(screen.getByLabelText("Credential"), { target: { value: "item-1" } });
    fireEvent.click(screen.getByRole("button", { name: "Create reviewed grant" }));
    await waitFor(() => expect(api.grants).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: /approve with fresh assurance/i }));
    await waitFor(() => expect(api.approve).toHaveBeenCalledWith("request-1", "digest-1", "assurance-token"));
    fireEvent.click(screen.getByRole("button", { name: "Deny" }));
    await waitFor(() => expect(api.deny).toHaveBeenCalledWith("request-1", "digest-1", "assurance-token"));
    fireEvent.change(screen.getByLabelText("Membership mode"), { target: { value: "dynamic" } });
    expect(screen.getByText(/Future members matching this principal/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Explain effective access" }));
    await waitFor(() => expect(api.effectiveAccess).toHaveBeenCalledWith("grant-1", "item-1", "use"));
    fireEvent.click(screen.getByRole("button", { name: /revoke and inspect residuals/i }));
    await waitFor(() => expect(api.revokeGrant).toHaveBeenCalledWith("grant-1"));

    fireEvent.click(screen.getByRole("button", { name: "Sources" }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Credential sources", level: 2 })).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Label"), { target: { value: "Backup authority" } });
    fireEvent.submit(screen.getByRole("button", { name: "Register source" }));
    expect(api.source).toHaveBeenCalledWith({ kind: "native", label: "Backup authority", endpoint: undefined, bootstrap_ref: undefined });

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    fireEvent.click(screen.getByRole("button", { name: "Connect browser" }));
    fireEvent.change(screen.getByLabelText("Owner token"), { target: { value: "owner-token" } });
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Vault action" })).not.toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: "Recovery" }));
    await waitFor(() => expect(screen.getByText("Recovery without guesswork")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Native export JSON"), { target: { value: "bundle" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview import" }));
    await waitFor(() => expect(api.preview).toHaveBeenCalledWith("bundle"));
    fireEvent.click(screen.getByRole("button", { name: "Commit import" }));
    await waitFor(() => expect(api.commit).toHaveBeenCalledWith("bundle", "skip"));

    fireEvent.click(screen.getByRole("button", { name: "Vault" }));
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    fireEvent.click(screen.getByText("GitHub"));
    fireEvent.click(screen.getByRole("button", { name: /edit after reveal/i }));
    await waitFor(() => expect(screen.getByRole("dialog", { name: "Edit credential" })).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "GitHub updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Save revision" }));
    await waitFor(() => expect(api.update).toHaveBeenCalled());
  });

  it("covers protected actions and provider controls without exposing values", async () => {
    defaults();
    api.login.mockResolvedValue({ access_token: "session-token" });
    api.sources.mockResolvedValue({ sources: [{ id: "source-1", kind: "onepassword", label: "Backup", status: "configured", endpoint: "https://vault.example", capabilities: { read: true } }] });
    api.sourceHealth.mockResolvedValue({ id: "source-1", kind: "onepassword", label: "Backup", status: "healthy", capabilities: { read: true } });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: /new item/i }));
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Generated login" } });
    fireEvent.change(screen.getByLabelText("Website origin"), { target: { value: "https://example.com" } });
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "owner@example.com" } });
    fireEvent.change(screen.getByLabelText("Private notes"), { target: { value: "private" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate" }));
    fireEvent.click(screen.getByRole("button", { name: "Save encrypted item" }));
    await waitFor(() => expect(api.create).toHaveBeenCalled());

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    const loginDialog = screen.getByRole("dialog", { name: "Sign in" });
    fireEvent.change(within(loginDialog).getByLabelText("Email"), { target: { value: "owner@example.com" } });
    fireEvent.change(within(loginDialog).getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(within(loginDialog).getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(api.login).toHaveBeenCalledWith("owner@example.com", "password"));

    fireEvent.click(screen.getByRole("button", { name: "First-owner setup" }));
    fireEvent.change(screen.getByLabelText("Bootstrap material"), { target: { value: "bootstrap" } });
    fireEvent.click(screen.getByRole("button", { name: "Enroll owner" }));
    await waitFor(() => expect(api.enroll).toHaveBeenCalledWith("bootstrap"));

    api.enroll.mockRejectedValue("invalid bootstrap material");
    fireEvent.click(screen.getByRole("button", { name: "First-owner setup" }));
    fireEvent.change(screen.getByLabelText("Bootstrap material"), { target: { value: "bad-bootstrap" } });
    fireEvent.click(screen.getByRole("button", { name: "Enroll owner" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Owner enrollment failed."));

    fireEvent.click(screen.getByRole("button", { name: "Access" }));
    await waitFor(() => expect(screen.getByText("Access that explains itself")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Principal type"), { target: { value: "human" } });
    fireEvent.change(screen.getByLabelText("Principal ID"), { target: { value: "human-1" } });
    fireEvent.change(screen.getByLabelText("Credential"), { target: { value: "item-1" } });
    fireEvent.change(screen.getByLabelText("Membership mode"), { target: { value: "dynamic" } });
    fireEvent.change(screen.getByLabelText("Allowed operation"), { target: { value: "reveal" } });
    fireEvent.change(screen.getByLabelText("Destination or origin"), { target: { value: "https://example.com" } });
    fireEvent.change(screen.getByLabelText("Duration"), { target: { value: "1" } });
    fireEvent.click(screen.getByRole("button", { name: "Create reviewed grant" }));
    await waitFor(() => expect(api.grants).toHaveBeenCalled());

    fireEvent.click(screen.getByRole("button", { name: "Activity" }));
    fireEvent.change(screen.getByLabelText("Filter activity"), { target: { value: "owner" } });
    fireEvent.change(screen.getByLabelText("Filter activity result"), { target: { value: "success" } });

    fireEvent.click(screen.getByRole("button", { name: "Sources" }));
    await waitFor(() => expect(screen.getByText("Backup")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Check health and capabilities" }));
    await waitFor(() => expect(api.sourceHealth).toHaveBeenCalledWith("source-1"));
  });

  it("keeps direct page routes and mobile list detail return context aligned", async () => {
    defaults();
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByText("GitHub")).toBeInTheDocument());
    fireEvent.click(screen.getByText("GitHub"));
    expect(screen.getByRole("button", { name: /back to vault list/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /back to vault list/i }));
    expect(screen.getByText("Select an item")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Recovery" }));
    await waitFor(() => expect(window.location.pathname).toBe("/recovery"));
    expect(screen.getByRole("heading", { name: "Recovery without guesswork", level: 2 })).toBeInTheDocument();
  });
});
