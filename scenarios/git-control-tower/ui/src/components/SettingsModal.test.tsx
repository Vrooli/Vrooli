import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsModal, type SettingsTab } from "./SettingsModal";
import { jsonResponse, renderWithQueryClient, setViewport } from "../test-utils";
import type { LayoutPreset, LayoutSection } from "./LayoutSettingsModal";
import type { GroupingRule } from "./FileList";

const connectMocks = vi.hoisted(() => ({
  repoClient: {
    listCredentials: vi.fn(),
    getPrecommitConfig: vi.fn(),
    savePrecommitConfig: vi.fn(),
  },
  humanControlClient: {
    getAuthorityStatus: vi.fn(),
    prepareMutation: vi.fn(),
    confirmMutation: vi.fn(),
  },
}));

vi.mock("../lib/connect", () => connectMocks);

function requestUrl(input: RequestInfo | URL) {
  if (input instanceof Request) return input.url;
  if (input instanceof URL) return input.toString();
  return input;
}

function settingsProps(overrides: Partial<React.ComponentProps<typeof SettingsModal>> = {}) {
  return {
    isOpen: true,
    repoDir: "/work/git-control-tower",
    repoId: "repo-1",
    syncStatus: {
      branch: "main",
      ahead: 0,
      behind: 0,
      has_upstream: true,
      can_push: false,
      can_pull: false,
      needs_push: false,
      needs_pull: false,
      has_uncommitted_changes: false,
      fetched: true,
      remote_url: "git@github.com:example/git-control-tower.git",
      timestamp: "2026-05-01T00:00:00Z",
    },
    preset: "classic" as LayoutPreset,
    primaryPanel: "changes" as LayoutSection,
    onChangePreset: vi.fn(),
    onChangePrimary: vi.fn(),
    onResetLayout: vi.fn(),
    groupingEnabled: true,
    onToggleGrouping: vi.fn(),
    groupingRules: [] as GroupingRule[],
    onChangeGroupingRules: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  };
}

describe("SettingsModal", () => {
  beforeEach(() => {
    setViewport(1280, 900);
    vi.clearAllMocks();
    connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true, reason: "" });
    connectMocks.humanControlClient.prepareMutation.mockResolvedValue({
      repositoryId: "repo-1",
      operation: "repo.precommit",
      expectedRevision: "head",
      subjectDigest: "digest",
    });
    connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  });

  it("does not render when closed", () => {
    renderWithQueryClient(<SettingsModal {...settingsProps({ isOpen: false })} />);

    expect(screen.queryByRole("dialog", { name: "Settings" })).not.toBeInTheDocument();
  });

  it("routes layout actions through the modal callbacks on desktop", () => {
    const props = settingsProps();

    renderWithQueryClient(<SettingsModal {...props} />);

    expect(screen.getByRole("dialog", { name: "Settings" })).toBeInTheDocument();
    expect(screen.getByText("Repo: /work/git-control-tower")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Split right" }));
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    fireEvent.click(screen.getByRole("button", { name: /reset to default/i }));
    fireEvent.click(screen.getByRole("button", { name: "Done" }));

    expect(props.onChangePreset).toHaveBeenCalledWith("split");
    expect(props.onChangePrimary).toHaveBeenCalledWith("review");
    expect(props.onResetLayout).toHaveBeenCalledOnce();
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it("switches to the integrations tab and renders capability status from the API", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) => {
      return jsonResponse({
        capabilities: [
          {
            id: "test-genie",
            name: "Test Genie",
            description: "Scenario test execution",
            dependencyKind: "scenario",
            dependencySlug: "test-genie",
            features: ["test runs", "phase diagnostics"],
            status: "available",
            message: "ready",
          },
          {
            id: "browser-automation-studio",
            name: "Browser Automation Studio",
            description: "Browser automation",
            dependencyKind: "scenario",
            dependencySlug: "browser-automation-studio",
            features: ["smoke"],
            status: "unavailable",
            message: "not running",
          },
        ],
        timestamp: "2026-05-01T00:00:00Z",
      });
    });
    connectMocks.repoClient.listCredentials.mockResolvedValue({
      credentials: [{
        id: "credential-1",
        remote: "origin",
        url: "https://github.com/example/git-control-tower.git",
        type: "https",
        username: "example",
        tokenMasked: "••••",
        isConfigured: true,
        createdAt: "2026-05-01T00:00:00Z",
        updatedAt: "2026-05-01T00:00:00Z",
      }],
      timestamp: "2026-05-01T00:00:00Z",
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(<SettingsModal {...settingsProps()} />);

    fireEvent.click(screen.getByRole("button", { name: "Integrations" }));

    // Counts the credential and both apps; only the running ones are ready.
    expect(await screen.findByText("2 of 3 ready")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "https://git-control-tower.test/api/v1/capabilities",
      expect.objectContaining({ cache: "no-store" }),
    );

    // Apps are named for what they do, and the one that is off is called out with its consequence.
    expect(screen.getByText("Tests")).toBeInTheDocument();
    expect(screen.getByText("Visual capture is off")).toBeInTheDocument();
    expect(screen.getByText("smoke won't work until Browser Automation Studio is running.")).toBeInTheDocument();
    // Git Control Tower cannot start other apps, so no fix button is offered.
    expect(screen.queryByRole("button", { name: /^start/i })).not.toBeInTheDocument();
    expect(await screen.findByText("github.com")).toBeInTheDocument();

    // A credential's detail view carries its controls and technical details.
    fireEvent.click(screen.getByRole("button", { name: /github\.com/ }));
    expect(await screen.findByRole("button", { name: "Test connection" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    expect(screen.getByText("https://github.com/example/git-control-tower.git")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));

    // An off app's detail view shows the command that starts it.
    fireEvent.click(screen.getByRole("button", { name: /^Visual capture/ }));
    expect(await screen.findByText("vrooli scenario start browser-automation-studio")).toBeInTheDocument();
  });

  it("reports a failed integrations check without hiding repository accounts", async () => {
    globalThis.fetch = vi.fn(async () => new Response("boom", { status: 500 })) as unknown as typeof fetch;
    connectMocks.repoClient.listCredentials.mockResolvedValue({ credentials: [], timestamp: "2026-05-01T00:00:00Z" });

    renderWithQueryClient(<SettingsModal {...settingsProps()} />);
    fireEvent.click(screen.getByRole("button", { name: "Integrations" }));

    expect(await screen.findByText(/Integrations can't be checked right now/)).toBeInTheDocument();
    expect(await screen.findByText(/No repository credentials yet/)).toBeInTheDocument();
  });

  it("treats a remote with nothing stored as using the system login, not as a problem", async () => {
    globalThis.fetch = vi.fn(async () => jsonResponse({ capabilities: [], timestamp: "2026-05-01T00:00:00Z" })) as unknown as typeof fetch;
    connectMocks.repoClient.listCredentials.mockResolvedValue({
      credentials: [{
        id: "origin",
        remote: "origin",
        url: "git@github.com:example/git-control-tower.git",
        type: "ssh",
        isConfigured: false,
        createdAt: "2026-05-01T00:00:00Z",
        updatedAt: "2026-05-01T00:00:00Z",
      }],
      timestamp: "2026-05-01T00:00:00Z",
    });

    renderWithQueryClient(<SettingsModal {...settingsProps()} />);
    fireEvent.click(screen.getByRole("button", { name: "Integrations" }));

    expect(await screen.findByText(/origin uses this computer's own Git login/)).toBeInTheDocument();
    expect(screen.queryByText(/isn't set up/)).not.toBeInTheDocument();
  });

  it("renders storage stats and sends a repo-scoped clear request", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (url.endsWith("/repo/visual-capture-storage") && init?.method === "DELETE") {
        return jsonResponse({});
      }
      return jsonResponse({
        totalSizeBytes: 3 * 1024 * 1024,
        snapshotCount: 3,
        perScenario: [
          {
            scenarioSlug: "git-control-tower",
            snapshotCount: 2,
            sizeBytes: 2 * 1024 * 1024,
          },
          {
            scenarioSlug: "workspace-sandbox",
            snapshotCount: 1,
            sizeBytes: 1024 * 1024,
          },
        ],
      });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <SettingsModal {...settingsProps({ initialTab: "storage" as SettingsTab })} />,
    );

    expect(await screen.findByText("3 snapshots — 3.0 MB")).toBeInTheDocument();
    expect(screen.getByText("git-control-tower")).toBeInTheDocument();
    expect(screen.getByText("workspace-sandbox")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /clear all snapshots/i }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "https://git-control-tower.test/api/v1/repo/visual-capture-storage",
        expect.objectContaining({
          method: "DELETE",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
        }),
      );
    });
  });

  it("renders the precommit tab; toggling Enabled auto-saves, and Save persists text-field edits", async () => {
    let serverState = {
      enabled: false,
      command: "",
      working_directory: "/work/git-control-tower",
      timeout_seconds: 300,
      run_before_commit: true,
      allow_override: true,
    };
    connectMocks.repoClient.getPrecommitConfig.mockImplementation(async () => ({
      enabled: serverState.enabled,
      command: serverState.command,
      workingDirectory: serverState.working_directory,
      timeoutSeconds: serverState.timeout_seconds,
      runBeforeCommit: serverState.run_before_commit,
      allowOverride: serverState.allow_override,
      timestamp: "2026-05-01T00:00:00Z",
    }));
    connectMocks.repoClient.savePrecommitConfig.mockImplementation(async (request: {
      enabled: boolean;
      command: string;
      workingDirectory: string;
      timeoutSeconds: number;
      runBeforeCommit: boolean;
      allowOverride: boolean;
    }) => {
      serverState = {
        enabled: request.enabled,
        command: request.command,
        working_directory: request.workingDirectory,
        timeout_seconds: request.timeoutSeconds,
        run_before_commit: request.runBeforeCommit,
        allow_override: request.allowOverride,
      };
      return {
        enabled: serverState.enabled,
        command: serverState.command,
        workingDirectory: serverState.working_directory,
        timeoutSeconds: serverState.timeout_seconds,
        runBeforeCommit: serverState.run_before_commit,
        allowOverride: serverState.allow_override,
        timestamp: "2026-05-01T00:00:00Z",
      };
    });

    renderWithQueryClient(
      <SettingsModal {...settingsProps({ initialTab: "precommit" as SettingsTab })} />,
    );

    expect(await screen.findByLabelText("Command")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Enabled"));
    await waitFor(() => {
      expect(serverState.enabled).toBe(true);
    });

    fireEvent.change(screen.getByLabelText("Command"), { target: { value: "make hygiene" } });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Save" })).not.toBeDisabled();
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(serverState.command).toBe("make hygiene");
    });
  });

  it("uses the library mobile drawer with one safe-area footer owner", () => {
    const props = settingsProps();
    setViewport(375, 740);

    renderWithQueryClient(<SettingsModal {...props} />);

    expect(screen.getByRole("dialog", { name: "Settings" })).toHaveClass("rcl-full-page-drawer__panel");
    expect(screen.getByText("Repo: /work/git-control-tower")).toBeInTheDocument();

    fireEvent.click(within(screen.getByRole("dialog", { name: "Settings" })).getByRole("button", { name: "Close" }));

    expect(props.onClose).toHaveBeenCalledOnce();
  });
});
