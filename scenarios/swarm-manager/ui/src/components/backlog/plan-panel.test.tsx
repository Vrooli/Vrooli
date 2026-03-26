import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PlanPanel } from "./plan-panel";

vi.mock("../../services", () => ({
  backlogService: {
    getFileContent: vi.fn(),
    saveFileContent: vi.fn(),
  },
}));

vi.mock("../../lib", async () => {
  const actual = await vi.importActual("../../lib");
  return {
    ...actual,
    defaultQueryOptions: { retry: false },
  };
});

vi.mock("@monaco-editor/react", () => ({
  default: ({ value, onChange }: { value: string; onChange?: (v: string) => void }) => (
    <textarea
      data-testid="mock-editor"
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
    />
  ),
}));

import { backlogService } from "../../services";

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

const mockPlanContent = "# Implementation Plan\n\nThis is the plan content.";

describe("PlanPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty state when plan.md is not found", async () => {
    vi.mocked(backlogService.getFileContent).mockRejectedValue(
      new Error("not found"),
    );

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByText("No plan yet")).toBeInTheDocument();
    });
  });

  it("renders plan content in rendered mode by default", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    expect(screen.getByLabelText("Raw editor")).toBeInTheDocument();
  });

  it("fetches plan.md via backlogService.getFileContent", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="fix" backlogName="my-fix" />,
    );

    await waitFor(() => {
      expect(backlogService.getFileContent).toHaveBeenCalledWith("fix", "my-fix", "plan.md");
    });
  });

  it("switches to raw editor mode when Edit button is clicked", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    expect(screen.getByTestId("mock-editor")).toBeInTheDocument();
    expect(screen.getByText("Save")).toBeInTheDocument();
    const saveBtn = screen.getByText("Save").closest("button");
    expect(saveBtn).toBeDisabled();
  });

  it("enables save button when content is edited", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    const editor = screen.getByTestId("mock-editor");
    fireEvent.change(editor, { target: { value: "edited content" } });

    await waitFor(() => {
      const saveBtn = screen.getByText("Save").closest("button");
      expect(saveBtn).not.toBeDisabled();
    });
  });

  it("calls saveFileContent on save", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);
    vi.mocked(backlogService.saveFileContent).mockResolvedValue({
      path: "plan.md",
      name: "plan.md",
      size: 100,
      type: "file",
    });

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
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
      expect(backlogService.saveFileContent).toHaveBeenCalledWith(
        "idea",
        "test-item",
        "plan.md",
        "edited content",
      );
    });
  });

  it("copies plan content to clipboard", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Copy"));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(mockPlanContent);
    });
  });

  it("shows discard button only when content is dirty in edit mode", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));
    expect(screen.queryByText("Discard")).not.toBeInTheDocument();

    const editor = screen.getByTestId("mock-editor");
    fireEvent.change(editor, { target: { value: "changed" } });

    await waitFor(() => {
      expect(screen.getByText("Discard")).toBeInTheDocument();
    });
  });

  it("restores original content on discard", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Raw editor"));

    const editor = screen.getByTestId("mock-editor");
    fireEvent.change(editor, { target: { value: "changed" } });

    await waitFor(() => {
      expect(screen.getByText("Discard")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Discard"));

    await waitFor(() => {
      expect((screen.getByTestId("mock-editor") as HTMLTextAreaElement).value).toBe(mockPlanContent);
    });
  });
});
