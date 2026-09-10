import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/clients", () => ({
  researchClient: {
    runL3: vi.fn(),
    waitResearch: vi.fn(),
    getEvidencePassage: vi.fn(),
  },
}));

import { researchClient } from "../../api/clients";
import { ResearchPanel } from "./ResearchPanel";

describe("ResearchPanel", () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
    vi.clearAllMocks();
  });

  it("persists the declared execution and resumes the same handle", async () => {
    vi.mocked(researchClient.runL3).mockResolvedValue({ runId: "run-1", status: "running" } as never);
    vi.mocked(researchClient.waitResearch).mockResolvedValue({ runId: "run-1", status: "complete", summary: "verified" } as never);
    renderWithProviders(<ResearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.research.query), { target: { value: "a verified question" } });
    fireEvent.click(screen.getByTestId(selectors.research.start));
    await waitFor(() => expect(researchClient.waitResearch).toHaveBeenCalledWith({ runId: "run-1", timeoutSeconds: 30 }));
    expect(window.localStorage.getItem("web-search.active-investigation")).toBe("run-1");
    fireEvent.click(screen.getByTestId(selectors.research.resume));
    await waitFor(() => expect(researchClient.waitResearch).toHaveBeenCalledTimes(2));
    expect(researchClient.waitResearch).toHaveBeenLastCalledWith({ runId: "run-1", timeoutSeconds: 30 });
  });

  it("renders readable evidence returned by an opaque passage lookup", async () => {
    vi.mocked(researchClient.getEvidencePassage).mockResolvedValue({ content: "The retained passage." } as never);
    renderWithProviders(<ResearchPanel />);
    fireEvent.change(screen.getByTestId(selectors.research.passageId), { target: { value: "passage-1" } });
    fireEvent.click(screen.getByTestId(selectors.research.readPassage));
    expect(await screen.findByTestId(selectors.research.passage)).toHaveTextContent("The retained passage.");
    expect(researchClient.getEvidencePassage).toHaveBeenCalledWith({ passageId: "passage-1" });
  });

  it("automatically resolves the first assessed evidence passage", async () => {
    vi.mocked(researchClient.waitResearch).mockResolvedValue({
      runId: "run-2",
      status: "complete",
      summary: "verified",
      result: { assessments: [{ evidence: [{ passage_id: "passage-auto" }] }] },
    } as never);
    vi.mocked(researchClient.getEvidencePassage).mockResolvedValue({ content: "Automatically loaded evidence." } as never);
    window.localStorage.setItem("web-search.active-investigation", "run-2");
    renderWithProviders(<ResearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.research.resume));
    await waitFor(() => expect(researchClient.waitResearch).toHaveBeenCalledWith({ runId: "run-2", timeoutSeconds: 30 }));
    expect(await screen.findByTestId(selectors.research.passage)).toHaveTextContent("Automatically loaded evidence.");
    expect(researchClient.getEvidencePassage).toHaveBeenCalledWith({ passageId: "passage-auto" });
  });

  it("exposes a partial result status and unresolved gaps", async () => {
    vi.mocked(researchClient.waitResearch).mockResolvedValue({
      runId: "run-partial",
      status: "complete",
      summary: "One required question remains unresolved.",
      result: { status: "partial", gaps: ["owner is unresolved"] },
    } as never);
    window.localStorage.setItem("web-search.active-investigation", "run-partial");
    renderWithProviders(<ResearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.research.resume));
    expect(await screen.findByTestId(selectors.research.resultStatus)).toHaveAttribute("data-result-status", "partial");
    expect(screen.getByTestId(selectors.research.gaps)).toHaveTextContent("owner is unresolved");
  });

  it("keeps the research controls keyboard and screen-reader accessible", async () => {
    vi.mocked(researchClient.waitResearch).mockResolvedValue({
      runId: "run-a11y",
      status: "complete",
      summary: "The result is partial; one required question remains unresolved.",
    } as never);
    window.localStorage.setItem("web-search.active-investigation", "run-a11y");
    const { container } = renderWithProviders(<ResearchPanel />);
    fireEvent.click(screen.getByTestId(selectors.research.resume));
    await waitFor(() => expect(researchClient.waitResearch).toHaveBeenCalled());
    await expectNoA11yViolations(container);
  });
});
