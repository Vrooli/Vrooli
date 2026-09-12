import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { RecordNarrativeEditor } from "./RecordNarrativeEditor";
import { renderWithProviders } from "../../test-utils";
import type { RecordItem } from "../../types";

function makeStub(overrides: Partial<RecordItem> = {}): RecordItem {
  return {
    id: "01HSTUB",
    kind: "fix",
    scenario: "audio-tools",
    trigger: "",
    approach: "",
    ruledOut: [],
    filesChanged: [],
    outcome: "shipped",
    stub: true,
    createdAt: "2026-05-27T12:00:00Z",
    ...overrides,
  };
}

describe("RecordNarrativeEditor", () => {
  it("refuses to edit a non-stub record", () => {
    renderWithProviders(
      <RecordNarrativeEditor record={makeStub({ stub: false, trigger: "x", approach: "y" })} onSubmit={vi.fn()} />,
    );
    expect(screen.queryByTestId("record-narrative-editor")).not.toBeInTheDocument();
    expect(screen.getByText(/already filled/i)).toBeInTheDocument();
  });

  it("calls onSubmit with parsed input on a stub", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(<RecordNarrativeEditor record={makeStub()} onSubmit={onSubmit} />);

    fireEvent.change(screen.getByTestId("record-trigger-input"), {
      target: { value: "voice auto-stop race" },
    });
    fireEvent.change(screen.getByTestId("record-approach-input"), {
      target: { value: "added silence_timed_out flag" },
    });
    fireEvent.change(screen.getByTestId("record-ruled-out-input"), {
      target: { value: "missing freshness check\nclient-only fix" },
    });

    fireEvent.click(screen.getByTestId("record-narrative-submit"));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        trigger: "voice auto-stop race",
        approach: "added silence_timed_out flag",
        ruledOut: ["missing freshness check", "client-only fix"],
        outcome: "shipped",
      }),
    );
  });

  it("rejects submission with empty required fields", async () => {
    const onSubmit = vi.fn();
    renderWithProviders(<RecordNarrativeEditor record={makeStub()} onSubmit={onSubmit} />);
    fireEvent.click(screen.getByTestId("record-narrative-submit"));
    expect(await screen.findByText(/Trigger and approach are required/i)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
