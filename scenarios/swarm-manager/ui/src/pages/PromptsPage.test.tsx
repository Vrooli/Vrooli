import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PromptsPage } from "./PromptsPage";

vi.mock("../services", () => ({
  promptService: {
    listBindings: vi.fn(),
    listSkills: vi.fn(),
    getSkill: vi.fn(),
    updateSkill: vi.fn(),
    getSkillVersions: vi.fn(),
    revertSkillVersion: vi.fn(),
    preview: vi.fn(),
    simulate: vi.fn(),
    getExecutionPromptTrace: vi.fn(),
    getBacklogPromptTrace: vi.fn(),
  },
}));

import { promptService } from "../services";

describe("PromptsPage", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.clearAllMocks();
    vi.mocked(promptService.listBindings).mockResolvedValue([
      {
        area: "research",
        trigger: "Backlog Research: Idea Clarify",
        kind: "idea",
        mode: "clarify",
        skill_id: "swarm-manager-clarify-idea",
        purpose: "Clarify scope",
      },
    ]);
    vi.mocked(promptService.listSkills).mockResolvedValue([
      {
        id: "swarm-manager-clarify-idea",
        name: "Clarify",
        description: "Clarify prompts",
        draft: false,
        trigger_count: 1,
        impact_summary: "Affects 1 trigger path(s).",
      },
    ]);
    vi.mocked(promptService.getSkill).mockResolvedValue({
      id: "swarm-manager-clarify-idea",
      name: "Clarify",
      description: "Clarify prompts",
      draft: false,
      trigger_count: 1,
      impact_summary: "Affects 1 trigger path(s).",
      current_content: "Use {{ITEM_TITLE}} in {{ITEM_FOLDER}}",
    });
    vi.mocked(promptService.getSkillVersions).mockResolvedValue({
      skillId: "swarm-manager-clarify-idea",
      current: 1,
      versions: [],
    });
  });

  it("renders map tab first and opens viewer tab when a prompt is clicked", async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <PromptsPage />
      </QueryClientProvider>
    );

    expect(await screen.findByTestId("prompts-page")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("prompts-tabs")).toBeInTheDocument();
      expect(screen.getByTestId("prompts-map-panel")).toBeInTheDocument();
      expect(screen.getByTestId("prompts-usage-matrix")).toBeInTheDocument();
      expect(screen.getByTestId("prompts-binding-map")).toBeInTheDocument();
    });
    expect(screen.getByTestId("prompts-viewer-panel")).not.toBeVisible();
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    expect(screen.getByText("Research")).toBeInTheDocument();
    expect(screen.getByText("Execution")).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: /Backlog Research: Idea Clarify/i })[0]);

    await waitFor(() => {
      expect(screen.getByTestId("prompts-viewer-panel")).toBeVisible();
      expect(screen.getByTestId("prompts-skills-list")).toBeInTheDocument();
      expect(screen.getByTestId("prompts-editor")).toBeInTheDocument();
    });
  });
});
