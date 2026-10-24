import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { GroupingRule } from "./FileList";
import { SettingsTabCredentials } from "./SettingsTabCredentials";
import { SettingsTabCredentialsSSH } from "./SettingsTabCredentialsSSH";
import { SettingsTabGrouping } from "./SettingsTabGrouping";
import { SettingsTabHealth } from "./SettingsTabHealth";
import { renderWithQueryClient } from "../test-utils";

const connectMocks = vi.hoisted(() => ({
  repoClient: {
    listCredentials: vi.fn(),
    saveCredential: vi.fn(),
    testCredential: vi.fn(),
    updateRemoteURL: vi.fn(),
    listSSHKeys: vi.fn(),
    getSSHPublicKey: vi.fn(),
    testSSHConnection: vi.fn(),
    generateSSHKey: vi.fn(),
    getGroupingRules: vi.fn(),
    getGitignoreHealth: vi.fn(),
    moveGitignoreEntry: vi.fn(),
    getTrackedBinaries: vi.fn(),
    untrackBinary: vi.fn(),
  },
  humanControlClient: {
    getAuthorityStatus: vi.fn(),
    prepareMutation: vi.fn(),
    confirmMutation: vi.fn(),
  },
}));

vi.mock("../lib/connect", () => connectMocks);

// AI_CHECK: GCT_TEST_ARCH=1 | LAST: 2026-05-01

describe("Settings tab surfaces", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
    connectMocks.repoClient.getGroupingRules.mockResolvedValue({
      enabled: true,
      rules: [{ id: "api", label: "API", prefixes: ["api/"], mode: "prefix" }],
    });
    connectMocks.repoClient.getGitignoreHealth.mockResolvedValue({
      rootEntryCount: 4,
      suggestions: [
        { type: "single_group", line: 12, pattern: "api/tmp/", groupDir: "api/", groupLabel: "API", targetPattern: "tmp/", hasGitignore: false },
        { type: "cross_group", line: 16, pattern: "*.log", groupDir: "", groupLabel: "workspace", targetPattern: "*.log", hasGitignore: true },
      ],
    });
    connectMocks.repoClient.getTrackedBinaries.mockResolvedValue({ binaries: [], totalBytes: 0, historyWarning: "" });
    connectMocks.repoClient.listCredentials.mockResolvedValue({ credentials: [{ remote: "origin", username: "octo", isConfigured: true, tokenMasked: "ghp_****", type: "https", url: "https://github.com/example/git-control-tower.git", createdAt: "", updatedAt: "" }], timestamp: "" });
    connectMocks.repoClient.saveCredential.mockResolvedValue({ success: true, timestamp: "" });
    connectMocks.repoClient.testCredential.mockResolvedValue({ success: true, authorized: true, reachable: true, timestamp: "" });
    connectMocks.repoClient.updateRemoteURL.mockResolvedValue({ success: true, newUrl: "git@github.com:example/git-control-tower.git", timestamp: "" });
    connectMocks.repoClient.listSSHKeys.mockResolvedValue({ keys: [], sshDir: "/home/user/.ssh", timestamp: "" });
    connectMocks.repoClient.getSSHPublicKey.mockResolvedValue({ success: true, publicKey: "ssh-ed25519 AAAA copied", timestamp: "" });
    connectMocks.repoClient.testSSHConnection.mockResolvedValue({ success: true, message: "Connected to GitHub", githubUser: "octo", timestamp: "" });
    connectMocks.repoClient.generateSSHKey.mockResolvedValue({ success: true, publicKey: "ssh-rsa BBBB generated", key: { path: "/home/user/.ssh/github_rsa", filename: "github_rsa", type: "rsa", fingerprint: "", hasPublic: true }, timestamp: "" });
    connectMocks.repoClient.moveGitignoreEntry.mockResolvedValue({ success: true });
    connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true });
    connectMocks.humanControlClient.prepareMutation.mockResolvedValue({ repositoryId: "repo-1", operation: "repo.gitignore.move", expectedRevision: "head", subjectDigest: "digest" });
    connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  });

  it("routes grouping rule edits through the supplied callbacks", () => {
    const onToggleGrouping = vi.fn();
    const onChangeRules = vi.fn();
    const rules: GroupingRule[] = [
      {
        id: "backend",
        label: "Backend",
        prefix: "api/",
        prefixes: ["api/"],
        mode: "prefix",
      },
    ];

    renderWithQueryClient(
      <SettingsTabGrouping
        groupingEnabled
        onToggleGrouping={onToggleGrouping}
        rules={rules}
        onChangeRules={onChangeRules}
        isMobile={false}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "On" }));
    expect(onToggleGrouping).toHaveBeenCalledOnce();

    fireEvent.change(screen.getByDisplayValue("Backend"), {
      target: { value: "API" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({ id: "backend", label: "API" }),
    ]);

    fireEvent.change(screen.getByDisplayValue("Prefix"), {
      target: { value: "segment" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({ id: "backend", mode: "segment" }),
    ]);

    fireEvent.change(screen.getByDisplayValue("api/"), {
      target: { value: "scenarios/" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: "backend",
        prefix: "scenarios/",
        prefixes: ["scenarios/"],
      }),
    ]);

    fireEvent.click(screen.getByRole("button", { name: /add prefix/i }));
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: "backend",
        prefix: "api/",
        prefixes: ["api/", ""],
      }),
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Remove group" }));
    expect(onChangeRules).toHaveBeenLastCalledWith([]);
  });

  it("shows contract groups as read-only rows", () => {
    renderWithQueryClient(
      <SettingsTabGrouping
        groupingEnabled
        onToggleGrouping={vi.fn()}
        rules={[]}
        onChangeRules={vi.fn()}
        isMobile={false}
        contractGroups={[{
          key: "contract:scenario:demo",
          kind: "scenario",
          id: "demo",
          label: "demo",
          root: "scenarios/demo",
          source: "contract",
          files: ["scenarios/demo/api/main.go"],
        }]}
      />,
    );

    expect(screen.getByTestId("contract-groups")).toBeInTheDocument();
    expect(screen.getByText("Contract-derived groups are read-only. Manual rules take precedence.")).toBeInTheDocument();
    expect(screen.getByText("demo")).toBeInTheDocument();
    expect(screen.getByText("scenarios/demo")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "demo" })).not.toBeInTheDocument();
  });

  it("renders gitignore health actions and sends repo-scoped move requests", async () => {
    renderWithQueryClient(<SettingsTabHealth isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/1 entry could be moved/i)).toBeInTheDocument();
    expect(screen.getByText("api/tmp/")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Move" }));

    await waitFor(async () => {
      expect(connectMocks.repoClient.moveGitignoreEntry).toHaveBeenCalledWith(expect.objectContaining({
        repositoryId: "repo-1",
        intentId: "intent-1",
        line: 12,
        pattern: "api/tmp/",
        groupDir: "api/",
        targetPattern: "tmp/",
      }));
    });

    fireEvent.click(screen.getByTitle("Dismiss"));
    expect(window.localStorage.getItem("gct.gitignore.dismissals")).toContain("api/tmp/");
    expect(screen.getByText("1 dismissed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /cross-group/i }));
    expect(screen.getByText("*.log")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /reset dismissals/i }));
    expect(window.localStorage.getItem("gct.gitignore.dismissals")).toBeNull();
  });

  it("saves HTTPS credentials, tests stored auth, and switches remote protocol", async () => {
    renderWithQueryClient(
      <SettingsTabCredentials
        remoteUrl="https://github.com/example/git-control-tower.git"
        hasUpstream
        isMobile={false}
        repoId="repo-1"
      />,
    );

    expect(await screen.findByText(/authenticated|connected/i)).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("GitHub username"), {
      target: { value: "octo" },
    });
    fireEvent.change(screen.getByPlaceholderText("ghp_xxxxxxxxxxxx"), {
      target: { value: "secret-token" },
    });
    fireEvent.click(screen.getByRole("button", { name: /save credentials/i }));

    await waitFor(() => expect(connectMocks.repoClient.saveCredential).toHaveBeenCalledWith(expect.objectContaining({
      repositoryId: "repo-1", remote: "origin", username: "octo", token: "secret-token",
    })));

    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("Connection successful!")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /switch to ssh/i }));
    await waitFor(() => expect(connectMocks.repoClient.updateRemoteURL).toHaveBeenCalledWith(expect.objectContaining({
      repositoryId: "repo-1", remote: "origin", url: "git@github.com:example/git-control-tower.git",
    })));
  });

  it("manages SSH key selection, key material, testing, and generation", async () => {
    const onCredentialsSaved = vi.fn();
    connectMocks.repoClient.listSSHKeys.mockResolvedValue({ keys: [
      { path: "/home/user/.ssh/github_ed25519", filename: "github_ed25519", type: "ed25519", fingerprint: "SHA256:abc", comment: "octo@example.com", createdAt: "2026-05-01T00:00:00Z", hasPublic: true },
      { path: "/home/user/.ssh/github_rsa", filename: "github_rsa", type: "rsa", fingerprint: "SHA256:def", comment: "", createdAt: "", hasPublic: true },
    ], sshDir: "/home/user/.ssh", timestamp: "" });

    renderWithQueryClient(
      <SettingsTabCredentialsSSH
        isMobile={false}
        repoId="repo-1"
        inputClasses="test-input"
        buttonHeight="h-8"
        storedSSHKeyPath="/home/user/.ssh/github_ed25519"
        onCredentialsSaved={onCredentialsSaved}
      />,
    );

    fireEvent.click(await screen.findByText("github_ed25519"));
    expect(screen.getByText("SHA256:abc")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /copy public key/i }));
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("ssh-ed25519 AAAA copied");
    });

    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("Connected to GitHub")).toBeInTheDocument();
    expect(screen.getByText("octo")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /save ssh key/i }));
    await waitFor(() => expect(connectMocks.repoClient.saveCredential).toHaveBeenCalledWith(expect.objectContaining({
      repositoryId: "repo-1", remote: "origin", sshKeyPath: "/home/user/.ssh/github_ed25519",
    })));
    expect(onCredentialsSaved).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: /generate new ssh key/i }));
    fireEvent.click(screen.getByRole("button", { name: /rsa/i }));
    fireEvent.change(screen.getByPlaceholderText("github_rsa"), {
      target: { value: "deploy_rsa" },
    });
    fireEvent.change(screen.getByPlaceholderText("your-email@example.com"), {
      target: { value: "deploy@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^generate key$/i }));

    expect(await screen.findByText("New SSH Key Generated!")).toBeInTheDocument();
    expect(screen.getByText("ssh-rsa BBBB generated")).toBeInTheDocument();

    expect(connectMocks.repoClient.generateSSHKey).toHaveBeenCalledWith(expect.objectContaining({
      repositoryId: "repo-1", type: "rsa", filename: "deploy_rsa", comment: "deploy@example.com",
    }));
  });
});
