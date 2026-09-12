import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { RecordCaptureForm } from "./RecordCaptureForm";
import { renderWithProviders } from "../../test-utils";
import type { RecordCaptureResult, RecordItem } from "../../types";

function draftResult(): RecordCaptureResult {
  return {
    disposition: "draft",
    record: { id: "rec-draft", kind: "fix", scenario: "swarm-manager", trigger: "", approach: "", ruledOut: [], filesChanged: [], outcome: "shipped", stub: false, draft: true, createdAt: "now" },
    accepted: { kind: "fix" }, needs: ["outcome", "trigger_or_approach_or_ruled_out"], invalid: [],
    warnings: ["Draft saved privately; it is not searchable or published."], nextAction: [],
  };
}

describe("RecordCaptureForm", () => {
  it("saves incomplete intake as a private draft and shows server repair needs", async () => {
    const submit = vi.fn().mockResolvedValue(draftResult());
    renderWithProviders(<RecordCaptureForm onSubmit={submit} />);

    fireEvent.change(screen.getByTestId("record-capture-scenario"), { target: { value: "swarm-manager" } });
    fireEvent.click(screen.getByTestId("record-capture-submit"));

    await waitFor(() => expect(submit).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("record-capture-disposition")).toHaveTextContent("Private draft saved");
    expect(screen.getByText(/outcome, trigger_or_approach_or_ruled_out/)).toBeInTheDocument();
    expect(screen.getByTestId("record-capture-open")).toHaveAttribute("href", "/records/rec-draft");
  });

  it("prefills a saved draft and submits a complete repair", async () => {
    const record: RecordItem = {
      ...draftResult().record,
      capture: { raw: { kind: "fix", scenario: "swarm-manager", trigger: "race", approach: "" }, needs: ["outcome"] },
    };
    const submit = vi.fn().mockResolvedValue({ ...draftResult(), disposition: "published", record: { ...record, draft: false } });
    renderWithProviders(<RecordCaptureForm record={record} onSubmit={submit} />);

    expect(screen.getByTestId("record-capture-trigger")).toHaveValue("race");
    fireEvent.change(screen.getByTestId("record-capture-approach"), { target: { value: "Fixed it" } });
    fireEvent.change(screen.getByTestId("record-capture-outcome"), { target: { value: "shipped" } });
    fireEvent.click(screen.getByTestId("record-capture-submit"));

    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ approach: "Fixed it", outcome: "shipped" })));
    expect(screen.getByTestId("record-capture-disposition")).toHaveTextContent("Published");
  });
});
