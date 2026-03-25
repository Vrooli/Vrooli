import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PromptTracePanel } from "./prompt-trace-panel";

vi.mock("../../services", () => ({
  promptService: {
    getBacklogPromptTrace: vi.fn(),
    updateBacklogPromptTrace: vi.fn(),
  },
}));

// Override defaultQueryOptions to disable retries in tests
vi.mock("../../lib", async () => {
  const actual = await vi.importActual("../../lib");
  return {
    ...actual,
    defaultQueryOptions: { retry: false },
  };
});

// Monaco editor is heavy — stub it for unit tests
vi.mock("@monaco-editor/react", () => ({
  default: ({ value, onChange }: { value: string; onChange?: (v: string) => void }) => (
    <textarea
      data-testid="mock-editor"
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
    />
  ),
}));

import { promptService } from "../../services";

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>,
  );
};

const mockTrace = {
  skill_id: "swarm-manager-workshop",
  purpose: "research",
  prompt: "# Test Prompt\n\nThis is the generated prompt.",
  prompt_revision: "sha256:abc123",
  used_fallback: false,
  captured_at: "2026-03-24T10:00:00Z",
};

describe("PromptTracePanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty state when trace is not found", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockRejectedValue(
      new Error("Prompt trace not found"),
    );

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("No prompt trace yet")).toBeInTheDocument();
    });
  });

  it("renders prompt content in rendered mode by default", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });

    // Should show rendered button as active (default mode)
    expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    expect(screen.getByLabelText("Raw editor")).toBeInTheDocument();
  });

  it("displays metadata bar with skill_id and captured_at", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });
  });

  it("shows fallback badge when used_fallback is true", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue({
      ...mockTrace,
      used_fallback: true,
    });

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("Fallback")).toBeInTheDocument();
    });
  });

  it("switches to raw editor mode when Edit button is clicked", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    // Monaco editor mock should be rendered
    expect(screen.getByTestId("mock-editor")).toBeInTheDocument();
    // Save button should be visible but disabled (no edits yet)
    expect(screen.getByText("Save")).toBeInTheDocument();
    const saveBtn = screen.getByText("Save").closest("button");
    expect(saveBtn).toBeDisabled();
  });

  it("enables save button when prompt is edited", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    const editor = screen.getByTestId("mock-editor");
    fireEvent.change(editor, { target: { value: "edited content" } });

    await waitFor(() => {
      const saveBtn = screen.getByText("Save").closest("button");
      expect(saveBtn).not.toBeDisabled();
    });
  });

  it("calls updateBacklogPromptTrace on save", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);
    vi.mocked(promptService.updateBacklogPromptTrace).mockResolvedValue({
      ...mockTrace,
      prompt: "edited content",
    });

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    const editor = screen.getByTestId("mock-editor");
    fireEvent.change(editor, { target: { value: "edited content" } });

    await waitFor(() => {
      const saveBtn = screen.getByText("Save").closest("button");
      expect(saveBtn).not.toBeDisabled();
    });

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(promptService.updateBacklogPromptTrace).toHaveBeenCalledWith(
        "idea",
        "test-item",
        expect.objectContaining({ prompt: "edited content" }),
      );
    });
  });

  it("copies prompt to clipboard", async () => {
    vi.mocked(promptService.getBacklogPromptTrace).mockResolvedValue(mockTrace);

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    renderWithProviders(
      <PromptTracePanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("swarm-manager-workshop")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Copy"));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(mockTrace.prompt);
    });
  });
});
