import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsModal, type SettingsTab } from "./SettingsModal";
import { jsonResponse, renderWithQueryClient, setViewport } from "../test-utils";
import type { LayoutPreset, LayoutSection } from "./LayoutSettingsModal";
import type { GroupingRule } from "./FileList";

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
    const fetchMock = vi.fn(async () => jsonResponse({
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
          id: "browserless",
          name: "Browserless",
          description: "Browser automation",
          dependencyKind: "resource",
          dependencySlug: "browserless",
          features: ["smoke"],
          status: "unavailable",
          message: "not running",
        },
      ],
      timestamp: "2026-05-01T00:00:00Z",
    }));
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(<SettingsModal {...settingsProps()} />);

    fireEvent.click(screen.getByRole("button", { name: "Integrations" }));

    expect(await screen.findByText("1/2 active")).toBeInTheDocument();
    expect(screen.getByText("Test Genie")).toBeInTheDocument();
    expect(screen.getByText("Browserless")).toBeInTheDocument();
    expect(screen.getByText("phase diagnostics")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "https://git-control-tower.test/api/v1/capabilities",
      expect.objectContaining({ cache: "no-store" }),
    );
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

  it("renders the precommit tab and saves repo-scoped settings", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (url.endsWith("/repo/precommit") && init?.method === "PUT") {
        return jsonResponse({
          enabled: true,
          command: "make hygiene",
          working_directory: "/work/git-control-tower",
          timeout_seconds: 300,
          run_before_commit: true,
          allow_override: true,
        });
      }
      return jsonResponse({
        enabled: false,
        command: "",
        working_directory: "/work/git-control-tower",
        timeout_seconds: 300,
        run_before_commit: true,
        allow_override: true,
      });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <SettingsModal {...settingsProps({ initialTab: "precommit" as SettingsTab })} />,
    );

    expect(await screen.findByLabelText("Command")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Enabled"));
    fireEvent.change(screen.getByLabelText("Command"), { target: { value: "make hygiene" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "https://git-control-tower.test/api/v1/repo/precommit",
        expect.objectContaining({
          method: "PUT",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
          body: expect.stringContaining("make hygiene"),
        }),
      );
    });
  });

  it("uses the full-screen mobile modal controls below the mobile breakpoint", () => {
    const props = settingsProps();
    setViewport(375, 740);

    renderWithQueryClient(<SettingsModal {...props} />);

    expect(screen.getByRole("dialog", { name: "Settings" })).toBeInTheDocument();
    expect(screen.getByText("Repo: /work/git-control-tower")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close" }));

    expect(props.onClose).toHaveBeenCalledOnce();
  });
});
