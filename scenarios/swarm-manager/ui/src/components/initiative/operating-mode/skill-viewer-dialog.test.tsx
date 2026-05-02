import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SkillViewerDialog } from "./skill-viewer-dialog";
import { selectors } from "../../../consts/selectors";
import { createQueryWrapper } from "../../../test-utils/query";

vi.mock("../../../config", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    dataFetchingConfig: {
      retryCount: 0,
      retryDelayMs: 0,
      staleTimeMs: 0,
      cacheTimeMs: 0,
      refetchOnWindowFocus: false,
    },
  };
});

vi.mock("../../../services/prompt-service", () => ({
  promptService: {
    getSkill: vi.fn(),
  },
}));

import { promptService } from "../../../services/prompt-service";

const sampleSkill = {
  id: "swarm-manager/holistic-loop-investigate",
  name: "Holistic Loop Investigate",
  description: "Surveys the initiative to find the right next slice.",
  draft: false,
  usage_type: "direct_runtime" as const,
  groups: ["initiative"],
  trigger_count: 3,
  impact_summary: "Used by 1 runtime path.",
  current_content: "# Investigate\n\nLook at **everything**.",
  required_missing: [],
};

function renderDialog(props: Partial<Parameters<typeof SkillViewerDialog>[0]> = {}) {
  return render(
    <SkillViewerDialog
      isOpen
      onClose={() => {}}
      skillId="swarm-manager/holistic-loop-investigate"
      {...props}
    />,
    { wrapper: createQueryWrapper() },
  );
}

describe("SkillViewerDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("does not render when closed", () => {
    renderDialog({ isOpen: false });
    expect(screen.queryByTestId(selectors.initiativeDetails.skillViewerDialog)).toBeNull();
  });

  it("renders the loading state while the skill is fetching", () => {
    vi.mocked(promptService.getSkill).mockImplementation(
      () => new Promise(() => {}),
    );
    renderDialog();
    expect(screen.getByTestId(selectors.initiativeDetails.skillViewerDialog)).toBeInTheDocument();
    expect(screen.getByText(/Loading skill content/)).toBeInTheDocument();
  });

  it("renders skill name, description, metadata chips, and rendered markdown body", async () => {
    vi.mocked(promptService.getSkill).mockResolvedValue(sampleSkill);
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText("Holistic Loop Investigate")).toBeInTheDocument();
    });
    expect(
      screen.getByText("Surveys the initiative to find the right next slice."),
    ).toBeInTheDocument();
    expect(screen.getByText("Direct runtime")).toBeInTheDocument();
    expect(screen.getByText("initiative")).toBeInTheDocument();
    // Markdown rendering: h1 from "#" prefix and bold from "**"
    expect(screen.getByRole("heading", { name: /Investigate/, level: 1 })).toBeInTheDocument();
    expect(screen.getByText("everything")).toBeInTheDocument();
  });

  it("renders an error block with a Retry button when the fetch fails", async () => {
    vi.mocked(promptService.getSkill).mockRejectedValue(new Error("upstream blew up"));
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText("upstream blew up")).toBeInTheDocument();
    });
    const retry = screen.getByTestId(selectors.initiativeDetails.skillViewerRetry);
    vi.mocked(promptService.getSkill).mockResolvedValue(sampleSkill);
    await userEvent.click(retry);
    await waitFor(() => {
      expect(screen.getByText("Surveys the initiative to find the right next slice.")).toBeInTheDocument();
    });
  });

  it("renders a draft chip when the skill is in draft", async () => {
    vi.mocked(promptService.getSkill).mockResolvedValue({ ...sampleSkill, draft: true });
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
    });
  });

  it("renders an empty-content fallback when the skill has no body", async () => {
    vi.mocked(promptService.getSkill).mockResolvedValue({ ...sampleSkill, current_content: "" });
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText(/no content body/)).toBeInTheDocument();
    });
  });

  it("copies the skill ID via the Copy ID button when clicked", async () => {
    vi.mocked(promptService.getSkill).mockResolvedValue(sampleSkill);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText("Holistic Loop Investigate")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.skillViewerCopyId));
    expect(writeText).toHaveBeenCalledWith("swarm-manager/holistic-loop-investigate");
    await waitFor(() => {
      expect(screen.getByText("Copied")).toBeInTheDocument();
    });
  });
});
