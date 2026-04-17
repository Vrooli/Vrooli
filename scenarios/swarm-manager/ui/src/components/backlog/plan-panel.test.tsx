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
  default: ({ value, onChange, onMount }: { value: string; onChange?: (v: string) => void; onMount?: (editor: unknown) => void }) => {
    // Call onMount with a mock editor on first render
    if (onMount) {
      const mockEditor = {
        revealLineInCenter: vi.fn(),
        setPosition: vi.fn(),
        focus: vi.fn(),
      };
      setTimeout(() => onMount(mockEditor), 0);
    }
    return (
      <textarea
        data-testid="mock-editor"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
    );
  },
}));

vi.mock("../../hooks/useModalBehavior", () => ({
  useModalBehavior: vi.fn(),
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

const mockPlanContent = "# Implementation Plan\n\nThis is the plan content.\n\n## Details\n\nSome details here.";

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

  it("renders icon-only buttons without text labels", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    // Buttons should be icon-only — no "Rendered", "Edit", or "Copy" text
    expect(screen.queryByText("Rendered")).not.toBeInTheDocument();
    expect(screen.queryByText("Edit")).not.toBeInTheDocument();
    expect(screen.queryByText("Copy")).not.toBeInTheDocument();
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
    const originalClipboard = navigator.clipboard;
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      writable: true,
      configurable: true,
    });

    renderWithProviders(
      <PlanPanel backlogKind="idea" backlogName="test-item" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Copy plan"));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(mockPlanContent);
    });

    // Restore original clipboard
    Object.defineProperty(navigator, "clipboard", {
      value: originalClipboard,
      writable: true,
      configurable: true,
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
      expect(screen.getByTestId("mock-editor")).toHaveValue(mockPlanContent);
    });
  });

  describe("TOC Popover", () => {
    it("shows TOC button when content has headings", async () => {
      vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

      renderWithProviders(
        <PlanPanel backlogKind="idea" backlogName="test-item" />,
      );

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });
    });

    it("does not show TOC button when content has no headings", async () => {
      vi.mocked(backlogService.getFileContent).mockResolvedValue("No headings here, just text.");

      renderWithProviders(
        <PlanPanel backlogKind="idea" backlogName="test-item" />,
      );

      await waitFor(() => {
        expect(screen.getByLabelText("Rendered view")).toBeInTheDocument();
      });

      expect(screen.queryByLabelText("Table of contents")).not.toBeInTheDocument();
    });

    it("opens TOC popover on click and shows heading entries", async () => {
      vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

      renderWithProviders(
        <PlanPanel backlogKind="idea" backlogName="test-item" />,
      );

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText("Table of contents"));

      expect(screen.getByTestId("toc-popover")).toBeInTheDocument();
      // Heading text appears in both the rendered content and the TOC popover
      const planEntries = screen.getAllByText("Implementation Plan");
      expect(planEntries.length).toBeGreaterThanOrEqual(2);
      const detailsEntries = screen.getAllByText("Details");
      expect(detailsEntries.length).toBeGreaterThanOrEqual(2);
    });

    it("closes TOC popover after clicking a heading entry", async () => {
      vi.mocked(backlogService.getFileContent).mockResolvedValue(mockPlanContent);

      // Mock scrollIntoView
      const scrollIntoView = vi.fn();
      const mockElement = { scrollIntoView };
      vi.spyOn(document, "getElementById").mockReturnValue(mockElement as unknown as HTMLElement);

      renderWithProviders(
        <PlanPanel backlogKind="idea" backlogName="test-item" />,
      );

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText("Table of contents"));
      // Click the TOC entry (button inside the nav), not the rendered heading
      const tocPopover = screen.getByTestId("toc-popover");
      const tocEntry = tocPopover.querySelector("button");
      expect(tocEntry).not.toBeNull();
      if (!tocEntry) {
        throw new Error("Expected a TOC entry button");
      }
      fireEvent.click(tocEntry);

      expect(screen.queryByTestId("toc-popover")).not.toBeInTheDocument();
    });
  });
});
