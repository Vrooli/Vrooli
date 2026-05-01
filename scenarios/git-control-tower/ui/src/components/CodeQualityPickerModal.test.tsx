import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  CodeQualityPickerModal,
  SEVERITY_LEVELS,
  CATEGORIES,
  LIMIT_PRESETS,
} from "./CodeQualityPickerModal";
import type { TidinessIssue } from "../lib/api";
import { renderWithQueryClient } from "../test-utils";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockUseTidinessIssues = vi.fn();

vi.mock("../lib/hooks", () => ({
  useTidinessIssues: (...args: unknown[]) => mockUseTidinessIssues(...args),
}));

vi.mock("../hooks", () => ({
  useIsMobile: () => false,
}));

// ---------------------------------------------------------------------------
// Factories
// ---------------------------------------------------------------------------

let nextId = 1;
function makeTidinessIssue(overrides?: Partial<TidinessIssue>): TidinessIssue {
  const id = nextId++;
  return {
    id,
    scenario: "test-scenario",
    file_path: `src/file${id}.ts`,
    category: "lint",
    severity: "medium",
    title: `Issue ${id}`,
    description: `Description for issue ${id}`,
    remediation_steps: "Fix it",
    status: "open",
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function makeIssueSet() {
  return [
    makeTidinessIssue({ severity: "high", category: "complexity", title: "High complexity" }),
    makeTidinessIssue({ severity: "medium", category: "length", title: "Long file" }),
    makeTidinessIssue({ severity: "low", category: "duplication", title: "Duplicated code" }),
  ];
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const defaultProps = {
  isOpen: true,
  onClose: vi.fn(),
  scenarioSlug: "test-scenario",
  repoId: null,
  onAttachItems: vi.fn(),
};

function renderModal(props = {}) {
  return renderWithQueryClient(<CodeQualityPickerModal {...defaultProps} {...props} />);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("CodeQualityPickerModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    nextId = 1;
    mockUseTidinessIssues.mockReturnValue({ data: [], isLoading: false });
  });

  it("does not render when isOpen is false", () => {
    renderModal({ isOpen: false });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders dialog when isOpen is true", () => {
    renderModal();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Code Quality Issues")).toBeInTheDocument();
  });

  it("renders issue list from hook data", () => {
    const issues = makeIssueSet();
    mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

    renderModal();
    expect(screen.getByText("High complexity")).toBeInTheDocument();
    expect(screen.getByText("Long file")).toBeInTheDocument();
    expect(screen.getByText("Duplicated code")).toBeInTheDocument();
  });

  it("shows empty state when no issues match", () => {
    mockUseTidinessIssues.mockReturnValue({ data: [], isLoading: false });
    renderModal();
    expect(screen.getByText("No issues match the current filters")).toBeInTheDocument();
  });

  it("shows loading skeleton", () => {
    mockUseTidinessIssues.mockReturnValue({ data: undefined, isLoading: true });
    renderModal();
    // Should not show empty state message while loading.
    expect(screen.queryByText("No issues match the current filters")).not.toBeInTheDocument();
  });

  it("renders all severity chips", () => {
    renderModal();
    for (const level of SEVERITY_LEVELS) {
      expect(
        screen.getByRole("button", { name: new RegExp(level, "i") }),
      ).toBeInTheDocument();
    }
  });

  it("renders all category chips plus All", () => {
    renderModal();
    expect(screen.getByRole("button", { name: "All" })).toBeInTheDocument();
    for (const cat of CATEGORIES) {
      // Category labels are human-friendly, not raw keys.
      const label =
        cat === "technical_debt" ? "Tech Debt" :
        cat === "type_safety" ? "Type Safety" :
        cat.charAt(0).toUpperCase() + cat.slice(1);
      expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
    }
  });

  it("renders limit preset buttons", () => {
    renderModal();
    for (const preset of LIMIT_PRESETS) {
      expect(screen.getByRole("button", { name: String(preset) })).toBeInTheDocument();
    }
  });

  describe("severity filtering", () => {
    it("hides issues when severity is toggled off", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      expect(screen.getByText("High complexity")).toBeInTheDocument();

      // Toggle off "high" severity — use aria-pressed to target the chip, not the issue row.
      const highChip = screen.getByRole("button", { name: /high/i, pressed: true });
      await user.click(highChip);
      expect(screen.queryByText("High complexity")).not.toBeInTheDocument();
      // Others remain.
      expect(screen.getByText("Long file")).toBeInTheDocument();
    });

    it("re-shows issues when severity is toggled back on", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      const highChip = screen.getByRole("button", { name: /high/i, pressed: true });
      await user.click(highChip);
      expect(screen.queryByText("High complexity")).not.toBeInTheDocument();

      const highChipOff = screen.getByRole("button", { name: /high/i, pressed: false });
      await user.click(highChipOff);
      expect(screen.getByText("High complexity")).toBeInTheDocument();
    });

    it("prevents deselecting the last severity", async () => {
      const user = userEvent.setup();
      const issues = [makeTidinessIssue({ severity: "high", title: "Only high" })];
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      // Deselect medium and low first.
      await user.click(screen.getByRole("button", { name: /medium/i, pressed: true }));
      await user.click(screen.getByRole("button", { name: /low/i, pressed: true }));
      // Try to deselect last one (high) — should be prevented.
      const highChip = screen.getByRole("button", { name: /high/i, pressed: true });
      await user.click(highChip);
      // High should still be pressed (deselection prevented).
      expect(highChip).toHaveAttribute("aria-pressed", "true");
      expect(screen.getByText("Only high")).toBeInTheDocument();
    });
  });

  describe("category filtering", () => {
    it("passes selected category to the hook", async () => {
      const user = userEvent.setup();
      renderModal();

      await user.click(screen.getByRole("button", { name: "Complexity" }));
      expect(mockUseTidinessIssues).toHaveBeenLastCalledWith(
        "test-scenario",
        expect.objectContaining({ category: "complexity" }),
      );
    });

    it("clears category when All is clicked", async () => {
      const user = userEvent.setup();
      renderModal();

      await user.click(screen.getByRole("button", { name: "Complexity" }));
      await user.click(screen.getByRole("button", { name: "All" }));
      expect(mockUseTidinessIssues).toHaveBeenLastCalledWith(
        "test-scenario",
        expect.objectContaining({ category: undefined }),
      );
    });
  });

  describe("limit presets", () => {
    it("passes selected limit to the hook", async () => {
      const user = userEvent.setup();
      renderModal();

      await user.click(screen.getByRole("button", { name: "10" }));
      expect(mockUseTidinessIssues).toHaveBeenLastCalledWith(
        "test-scenario",
        expect.objectContaining({ limit: 10 }),
      );
    });
  });

  describe("issue selection", () => {
    it("auto-selects all issues on load", () => {
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      const dialog = screen.getByRole("dialog");
      // The attach button should show count matching all issues.
      expect(within(dialog).getByText(`Attach ${issues.length} selected`)).toBeInTheDocument();
    });

    it("deselects an issue when clicked", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      // Click the first issue row to deselect it.
      await user.click(screen.getByText("High complexity"));
      expect(screen.getByText(`Attach ${issues.length - 1} selected`)).toBeInTheDocument();
    });

    it("select all re-selects all issues", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      // Deselect one.
      await user.click(screen.getByText("High complexity"));
      // Click "Select all".
      await user.click(screen.getByText("Select all"));
      expect(screen.getByText(`Attach ${issues.length} selected`)).toBeInTheDocument();
    });

    it("clear deselects all issues", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      await user.click(screen.getByText("Clear"));
      expect(screen.getByText("Attach 0 selected")).toBeInTheDocument();
    });
  });

  describe("attach", () => {
    it("calls onAttachItems with selected issues and closes", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      const onAttachItems = vi.fn();
      const onClose = vi.fn();
      renderModal({ onAttachItems, onClose });

      // Deselect the last issue to ensure only first two are attached.
      await user.click(screen.getByText("Duplicated code"));

      await user.click(screen.getByText(`Attach ${issues.length - 1} selected`));

      expect(onAttachItems).toHaveBeenCalledTimes(1);
      const attachedItems = onAttachItems.mock.calls[0]?.[0];
      expect(attachedItems).toHaveLength(issues.length - 1);
      expect(attachedItems[0].kind).toBe("code-quality-issue");
      expect(onClose).toHaveBeenCalled();
    });

    it("disables attach button when no issues selected", async () => {
      const user = userEvent.setup();
      const issues = makeIssueSet();
      mockUseTidinessIssues.mockReturnValue({ data: issues, isLoading: false });

      renderModal();
      await user.click(screen.getByText("Clear"));
      const attachButton = screen.getByText("Attach 0 selected").closest("button");
      expect(attachButton).toBeDisabled();
    });
  });

  describe("close", () => {
    it("calls onClose when X button is clicked", async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      renderModal({ onClose });

      await user.click(screen.getByLabelText("Close"));
      expect(onClose).toHaveBeenCalled();
    });
  });

  describe("clear filters", () => {
    it("resets all filter state", async () => {
      const user = userEvent.setup();
      renderModal();

      // Change some filters.
      await user.click(screen.getByRole("button", { name: /high/i }));
      await user.click(screen.getByRole("button", { name: "Complexity" }));
      await user.click(screen.getByRole("button", { name: "10" }));

      // Clear filters.
      await user.click(screen.getByText("Clear filters"));

      // Should reset to defaults: all severities, no category, limit 25.
      expect(mockUseTidinessIssues).toHaveBeenLastCalledWith(
        "test-scenario",
        expect.objectContaining({ category: undefined, limit: 25 }),
      );
    });
  });

  it("exports constants for external use", () => {
    expect(SEVERITY_LEVELS).toEqual(["high", "medium", "low"]);
    expect(CATEGORIES).toEqual([
      "length", "complexity", "duplication",
      "technical_debt", "coupling", "type_safety",
    ]);
    expect(LIMIT_PRESETS).toEqual([10, 25, 50]);
  });
});
