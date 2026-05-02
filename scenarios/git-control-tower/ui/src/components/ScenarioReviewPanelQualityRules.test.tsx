import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CodeQualityTab } from "./ScenarioReviewPanelCodeQuality";
import { RulesTab } from "./ScenarioReviewPanelRules";
import { renderWithQueryClient, jsonResponse } from "../test-utils";
import type { AuditorJobStatus, RepoFileStats, TidinessIssue } from "../lib/api";

function requestUrl(input: RequestInfo | URL) {
  if (input instanceof Request) return input.url;
  if (input instanceof URL) return input.toString();
  return input;
}

function changedFileStats(): RepoFileStats {
  return {
    staged: {
      "scenarios/git-control-tower/ui/src/App.tsx": {
        additions: 12,
        deletions: 3,
        files: 1,
      },
    },
    unstaged: {
      "scenarios/git-control-tower/api/main.go": {
        additions: 4,
        deletions: 1,
        files: 1,
      },
    },
  };
}

function tidinessIssue(overrides: Partial<TidinessIssue> = {}): TidinessIssue {
  return {
    id: 41,
    scenario: "git-control-tower",
    file_path: "ui/src/App.tsx",
    category: "complexity",
    severity: "high",
    title: "Component does too much",
    description: "Split orchestration from rendering.",
    line_number: 27,
    status: "open",
    created_at: "2026-05-01T12:00:00Z",
    ...overrides,
  };
}

function completedRulesJob(): AuditorJobStatus {
  return {
    id: "rules-job-1",
    scenario: "git-control-tower",
    scan_type: "full",
    status: "completed",
    started_at: "2026-05-01T12:00:00Z",
    completed_at: "2026-05-01T12:00:02Z",
    elapsed_seconds: 2,
    total_scenarios: 1,
    processed_scenarios: 1,
    processed_files: 17,
    total_files: 17,
    result: {
      check_id: "check-1",
      status: "completed",
      scan_type: "full",
      started_at: "2026-05-01T12:00:00Z",
      completed_at: "2026-05-01T12:00:02Z",
      duration_seconds: 2,
      files_scanned: 17,
      statistics: {},
      message: "Completed",
      summary: {
        total: 1,
        by_severity: { high: 1 },
        highest_severity: "high",
        recommended_steps: ["Fix required structure before release."],
        generated_at: "2026-05-01T12:00:02Z",
      },
      violations: [
        {
          id: "required-layout",
          scenario_name: "git-control-tower",
          type: "required_layout",
          severity: "high",
          title: "Scenario Required Structure",
          description: "Makefile must expose standard lifecycle targets.",
          file_path: "Makefile",
          line_number: 23,
          code_snippet: "test:",
          recommendation: "Add standard lifecycle wrapper targets.",
          standard: "scenario",
          discovered_at: "2026-05-01T12:00:02Z",
          source: "scenario-auditor",
        },
      ],
    },
  };
}

describe("CodeQualityTab", () => {
  it("matches tidiness issues against scenario-relative changed paths", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = requestUrl(input);
      if (url.includes("/repo/tidiness-score?")) {
        return jsonResponse({
          scenario: "git-control-tower",
          score: 66,
          violations: 1,
          metrics: { total_files: 8 },
        });
      }
      if (url.includes("/repo/tidiness-issues?")) {
        return jsonResponse([
          tidinessIssue(),
          tidinessIssue({ id: 42, file_path: "ui/src/Unchanged.tsx", title: "Unchanged issue" }),
        ]);
      }
      if (url.includes("/repo/tidiness-staleness?")) {
        return jsonResponse({
          last_scan_at: "2026-05-01T12:00:00Z",
          is_stale: false,
        });
      }
      return jsonResponse({});
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;
    const onAttachToAgent = vi.fn();

    renderWithQueryClient(
      <CodeQualityTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        tidinessAvailable
        fileStats={changedFileStats()}
        agentManagerAvailable
        onAttachToAgent={onAttachToAgent}
      />,
    );

    expect(await screen.findByText("ui/src/App.tsx")).toBeInTheDocument();
    expect(screen.getByText("(1 issue)")).toBeInTheDocument();
    expect(screen.queryByText("ui/src/Unchanged.tsx")).not.toBeInTheDocument();

    fireEvent.click(screen.getByText("ui/src/App.tsx"));
    expect(screen.getByText("Component does too much")).toBeInTheDocument();

    const attachButtons = screen.getAllByRole("button", { name: /\+ agent/i });
    const attachButton = attachButtons[0];
    if (!attachButton) throw new Error("expected an agent attach button");
    fireEvent.click(attachButton);
    expect(onAttachToAgent).toHaveBeenCalledWith(expect.objectContaining({
      kind: "code-quality-issue",
    }));
  });

  it("runs a first scan from the never-scanned state", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (init?.method === "POST" && url.endsWith("/repo/tidiness-scan")) {
        return jsonResponse({
          scenario: "git-control-tower",
          started_at: "2026-05-01T12:00:00Z",
          completed_at: "2026-05-01T12:00:01Z",
          duration_ms: 1000,
          file_metrics: [],
          long_files: [],
          total_files: 8,
          total_lines: 1200,
          lint_issues: 0,
          type_issues: 0,
          long_files_count: 0,
        });
      }
      if (url.includes("/repo/tidiness-staleness?")) {
        return jsonResponse({
          is_stale: true,
          stale_reason: "no scans have been run",
        });
      }
      return jsonResponse([]);
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <CodeQualityTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        tidinessAvailable
      />,
    );

    fireEvent.click(await screen.findByRole("button", { name: /run scan/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "https://git-control-tower.test/api/v1/repo/tidiness-scan",
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
          body: JSON.stringify({ scenario_name: "git-control-tower", incremental: false }),
        }),
      );
    });
  });
});

describe("RulesTab", () => {
  it("starts a standards check and persists the returned job id", async () => {
    const onJobIdChange = vi.fn();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (init?.method === "POST" && url.endsWith("/repo/rules-run")) {
        return jsonResponse({
          job_id: "rules-job-1",
          status: {
            id: "rules-job-1",
            scenario: "git-control-tower",
            scan_type: "full",
            status: "pending",
            started_at: "2026-05-01T12:00:00Z",
            elapsed_seconds: 0,
            total_scenarios: 1,
            processed_scenarios: 0,
            processed_files: 0,
            total_files: 10,
          },
        });
      }
      if (url.includes("/repo/rules-job/rules-job-1")) {
        return jsonResponse(completedRulesJob());
      }
      return jsonResponse({});
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <RulesTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        auditorAvailable
        onJobIdChange={onJobIdChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /run check/i }));

    await waitFor(() => {
      expect(onJobIdChange).toHaveBeenCalledWith("rules-job-1");
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "https://git-control-tower.test/api/v1/repo/rules-run",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
        body: JSON.stringify({ scenario_name: "git-control-tower", check_type: "full" }),
      }),
    );
  });

  it("renders completed violations, expansion details, and agent context actions", async () => {
    const onAttachToAgent = vi.fn();
    const fetchMock = vi.fn(async () => jsonResponse(completedRulesJob()));
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <RulesTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        auditorAvailable
        agentManagerAvailable
        onAttachToAgent={onAttachToAgent}
        initialJobId="rules-job-1"
      />,
    );

    expect(await screen.findByText("1 violation")).toBeInTheDocument();
    expect(screen.getByText("17 files scanned")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Scenario Required Structure"));

    expect(screen.getByText("Makefile must expose standard lifecycle targets.")).toBeInTheDocument();
    expect(screen.getByText("Makefile:23")).toBeInTheDocument();
    expect(screen.getByText("Add standard lifecycle wrapper targets.")).toBeInTheDocument();

    const attachButtons = screen.getAllByRole("button", { name: /\+ agent/i });
    const attachButton = attachButtons[0];
    if (!attachButton) throw new Error("expected an agent attach button");
    fireEvent.click(attachButton);
    expect(onAttachToAgent).toHaveBeenCalledWith(expect.objectContaining({
      kind: "rules-summary",
    }));
  });
});
