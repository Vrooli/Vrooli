import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

const retrieval = vi.hoisted(() => ({ ambient: vi.fn() }));
const triage = vi.hoisted(() => ({ getTriage: vi.fn(), setDisposition: vi.fn(), addAnnotation: vi.fn() }));

vi.mock("../../api/retrieval", () => ({ retrievalClient: retrieval }));
vi.mock("../../api/triage", () => ({ triageClient: triage }));
vi.mock("../categories/SignalClassificationControl", () => ({ SignalClassificationControl: () => <span>Advisory classification</span> }));

import { TriageQueue } from "./TriageQueue";

describe("TriageQueue [REQ:SIG-P0-013]", () => {
  beforeEach(() => {
    retrieval.ambient.mockResolvedValue({ results: [{ signal: { id: "signal-1", sourceKind: "url", sourceUrl: "https://example.test", extractedContent: "Useful extracted content", needsAttention: false }, disposition: "new" }] });
    triage.getTriage.mockResolvedValue({ triage: { annotations: [{ id: "annotation-1", author: 1, body: "Operator context" }] } });
    triage.setDisposition.mockResolvedValue({});
    triage.addAnnotation.mockResolvedValue({});
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("renders one unresolved signal and records a non-destructive drop", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TriageQueue />);
    expect(await screen.findByText("Useful extracted content")).toBeInTheDocument();
    expect(await screen.findByText("Operator context")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /drop from ambient/i }));
    await waitFor(() => expect(triage.setDisposition).toHaveBeenCalledWith({ signalId: "signal-1", state: 5 }));
  });

  it("accepts and annotates through keyboard shortcuts", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TriageQueue />);
    await screen.findByText("Useful extracted content");
    fireEvent.keyDown(window, { key: "a" });
    await waitFor(() => expect(triage.setDisposition).toHaveBeenCalledWith({ signalId: "signal-1", state: 2 }));
    await user.type(screen.getByLabelText("Triage annotation"), "Carry into planning");
    fireEvent.keyDown(window, { key: "n" });
    await waitFor(() => expect(triage.addAnnotation).toHaveBeenCalledWith({ signalId: "signal-1", author: 1, body: "Carry into planning" }));
  });

  it("renders the clear-queue state", async () => {
    retrieval.ambient.mockResolvedValue({ results: [] });
    renderWithProviders(<TriageQueue />);
    expect(await screen.findByText(/queue is clear/i)).toBeInTheDocument();
  });
});
