import { renderWithProviders as render } from "../../../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import HandoffRulesPanel from "../HandoffRulesPanel";
import { deleteHandoffRule, listHandoffRules, upsertHandoffRule } from "../../../api/handoffrules";

vi.mock("../../../api/handoffrules", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../../api/handoffrules")>()),
  deleteHandoffRule: vi.fn(),
  listHandoffRules: vi.fn(),
  upsertHandoffRule: vi.fn(),
}));

const rule = { id: "rule-1", name: "Markdown", enabled: true, source: "file_path" as const, pattern: "**/*.md", surfaces: ["messages"], sort_order: 0 };

describe("HandoffRulesPanel", () => {
  beforeEach(() => {
    vi.mocked(listHandoffRules).mockReset();
    vi.mocked(upsertHandoffRule).mockReset();
    vi.mocked(deleteHandoffRule).mockReset();
  });

  it("shows an explicit empty state and creates a safe disabled rule", async () => {
    vi.mocked(listHandoffRules).mockResolvedValue([]);
    vi.mocked(upsertHandoffRule).mockResolvedValue({ ...rule, id: "new-rule", enabled: false, name: "New rule" });
    render(<HandoffRulesPanel />);
    expect(await screen.findByTestId("handoff-rules-empty")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("handoff-rules-create"));
    await waitFor(() => expect(upsertHandoffRule).toHaveBeenCalledWith(expect.objectContaining({ enabled: false, source: "file_path", pattern: "**/*.md" })));
  });

  it("optimistically edits, toggles, changes source, and removes a rule", async () => {
    vi.mocked(listHandoffRules).mockResolvedValue([rule]);
    vi.mocked(upsertHandoffRule).mockResolvedValue(rule);
    render(<HandoffRulesPanel />);
    await screen.findByTestId("handoff-rule-rule-1");
    fireEvent.click(screen.getByTestId("handoff-rule-toggle-rule-1"));
    fireEvent.change(screen.getByTestId("handoff-rule-source-rule-1"), { target: { value: "message_text" } });
    fireEvent.change(screen.getByTestId("handoff-rule-name-rule-1"), { target: { value: "Messages" } });
    await waitFor(() => expect(upsertHandoffRule).toHaveBeenCalled());
    fireEvent.click(screen.getByTestId("handoff-rule-delete-rule-1"));
    await waitFor(() => expect(deleteHandoffRule).toHaveBeenCalledWith("rule-1"));
  });
});
