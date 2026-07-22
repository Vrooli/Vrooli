import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PromptsPage } from "./PromptsPage";

vi.mock("../services", () => ({
  promptService: {
    listCatalog: vi.fn(),
    listSkills: vi.fn(),
    getSkill: vi.fn(),
    updateSkill: vi.fn(),
    getSkillVersions: vi.fn(),
    revertSkillVersion: vi.fn(),
    preview: vi.fn(),
    getExecutionPromptTrace: vi.fn(),
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
    vi.mocked(promptService.listCatalog).mockResolvedValue([
      {
        id: "capture-classify",
        title: "Capture Classification",
        group: "capture",
        usage_type: "direct_runtime",
        source_type: "skill",
        trigger: "Capture classify action",
        skill_id: "swarm-manager-classify-capture",
        purpose: "Analyze a capture and suggest backlog items.",
      },
      {
        id: "execution-process",
        title: "Execution Process Prompt",
        group: "execution",
        usage_type: "generated_runtime",
        source_type: "generated",
        trigger: "Execution start / retry",
        builder: "execution.buildExecutionPrompt",
        operations: ["generator", "improver"],
        purpose: "Build the runtime execution prompt from the deliverable.",
      },
    ]);
    vi.mocked(promptService.listSkills).mockResolvedValue([
      {
        id: "swarm-manager-classify-capture",
        name: "Capture Classification",
        description: "Capture classification prompts",
        draft: false,
        usage_type: "direct_runtime",
        groups: ["capture"],
        trigger_count: 1,
        impact_summary: "Used directly by 1 runtime prompt path.",
      },
    ]);
    vi.mocked(promptService.getSkill).mockResolvedValue({
        id: "swarm-manager-classify-capture",
        name: "Capture Classification",
        description: "Capture classification prompts",
      draft: false,
      usage_type: "direct_runtime",
        groups: ["capture"],
      trigger_count: 1,
      impact_summary: "Used directly by 1 runtime prompt path.",
      current_content: "Use {{ITEM_TITLE}} in {{ITEM_FOLDER}}",
    });
    vi.mocked(promptService.getSkillVersions).mockResolvedValue({
      skillId: "swarm-manager-classify-capture",
      current: 1,
      versions: [],
    });
  });

  it("renders catalog tab first and opens viewer tab when a skill-backed entry is clicked", async () => {
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
    expect(screen.getByText("Capture")).toBeInTheDocument();
    expect(screen.getByText("Backlog")).toBeInTheDocument();
    expect(screen.getByText("Execution")).toBeInTheDocument();
    expect(screen.getByText("Prompt Catalog")).toBeInTheDocument();

    const catalogButtons = screen.getAllByRole("button", { name: /Capture Classification/i });
    expect(catalogButtons.length).toBeGreaterThan(0);
    const firstCatalogButton = catalogButtons[0];
    if (!firstCatalogButton) {
      throw new Error("Expected at least one prompt catalog button");
    }
    fireEvent.click(firstCatalogButton);

    await waitFor(() => {
      expect(screen.getByTestId("prompts-viewer-panel")).toBeVisible();
      expect(screen.getAllByTestId("prompts-skills-list").length).toBeGreaterThan(0);
      expect(screen.getByTestId("prompts-editor")).toBeInTheDocument();
    });
  });
});
