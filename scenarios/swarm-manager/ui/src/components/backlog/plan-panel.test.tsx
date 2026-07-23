import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PlanPanel } from "./plan-panel";
import { ApiError } from "../../lib/api-client";

vi.mock("../../services", () => ({
  backlogService: {
    getRenderedPlan: vi.fn(),
  },
}));

vi.mock("../../lib", async () => {
  const actual = await vi.importActual("../../lib");
  return {
    ...actual,
    defaultQueryOptions: { retry: false },
  };
});

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
const mockRenderedPlan = {
  path: "plan-manager:test-plan",
  markdown: mockPlanContent,
  qualityStatus: "clean",
  qualityFindings: [],
  planRef: {
    provider: "plan-manager" as const,
    planId: "plan-1",
    slug: "test-plan",
    role: "execution_spec" as const,
  },
};

describe("PlanPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty state when no linked plan is found", async () => {
    vi.mocked(backlogService.getRenderedPlan).mockRejectedValue(new ApiError("http", "Plan not found", { status: 404, code: "plan_ref_not_found" }));

    renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

    await waitFor(() => {
      expect(screen.getByText("No plan yet")).toBeInTheDocument();
    });
  });

  it("offers a direct plan-author action when a plan is absent", async () => {
    vi.mocked(backlogService.getRenderedPlan).mockRejectedValue(new ApiError("http", "Plan not found", { status: 404, code: "plan_ref_not_found" }));
    const onAuthorPlan = vi.fn();

    renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" onAuthorPlan={onAuthorPlan} />);

    await waitFor(() => expect(screen.getByTestId("backlog-plan-author-cta")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("backlog-plan-author-cta"));
    expect(onAuthorPlan).toHaveBeenCalledTimes(1);
  });

  it("fetches and renders canonical plan-manager markdown", async () => {
    vi.mocked(backlogService.getRenderedPlan).mockResolvedValue(mockRenderedPlan);

    renderWithProviders(<PlanPanel backlogKind="fix" backlogName="my-fix" />);

    await waitFor(() => {
      expect(backlogService.getRenderedPlan).toHaveBeenCalledWith("fix", "my-fix");
      expect(screen.getByLabelText("Copy plan")).toBeInTheDocument();
    });

    expect(screen.getByText("plan-manager:test-plan")).toBeInTheDocument();
    expect(screen.getByLabelText("Open in plan-manager")).toBeInTheDocument();
  });

  it("copies rendered plan content to clipboard", async () => {
    vi.mocked(backlogService.getRenderedPlan).mockResolvedValue(mockRenderedPlan);

    const writeText = vi.fn().mockResolvedValue(undefined);
    const originalClipboard = navigator.clipboard;
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      writable: true,
      configurable: true,
    });

    renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

    await waitFor(() => {
      expect(screen.getByLabelText("Copy plan")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Copy plan"));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(mockPlanContent);
    });

    Object.defineProperty(navigator, "clipboard", {
      value: originalClipboard,
      writable: true,
      configurable: true,
    });
  });

  describe("TOC Popover", () => {
    it("shows TOC button when content has headings", async () => {
      vi.mocked(backlogService.getRenderedPlan).mockResolvedValue(mockRenderedPlan);

      renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });
    });

    it("does not show TOC button when content has no headings", async () => {
      vi.mocked(backlogService.getRenderedPlan).mockResolvedValue({
        ...mockRenderedPlan,
        markdown: "No headings here, just text.",
      });

      renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

      await waitFor(() => {
        expect(screen.getByLabelText("Copy plan")).toBeInTheDocument();
      });

      expect(screen.queryByLabelText("Table of contents")).not.toBeInTheDocument();
    });

    it("opens TOC popover on click and shows heading entries", async () => {
      vi.mocked(backlogService.getRenderedPlan).mockResolvedValue(mockRenderedPlan);

      renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText("Table of contents"));

      expect(screen.getByTestId("toc-popover")).toBeInTheDocument();
      expect(screen.getAllByText("Implementation Plan").length).toBeGreaterThanOrEqual(2);
      expect(screen.getAllByText("Details").length).toBeGreaterThanOrEqual(2);
    });

    it("closes TOC popover after clicking a heading entry", async () => {
      vi.mocked(backlogService.getRenderedPlan).mockResolvedValue(mockRenderedPlan);

      const scrollIntoView = vi.fn();
      vi.spyOn(document, "getElementById").mockReturnValue({ scrollIntoView } as unknown as HTMLElement);

      renderWithProviders(<PlanPanel backlogKind="idea" backlogName="test-item" />);

      await waitFor(() => {
        expect(screen.getByLabelText("Table of contents")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText("Table of contents"));
      const tocEntry = screen.getByTestId("toc-popover").querySelector("button");
      expect(tocEntry).not.toBeNull();
      fireEvent.click(tocEntry as HTMLButtonElement);

      expect(screen.queryByTestId("toc-popover")).not.toBeInTheDocument();
    });
  });
});
