import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { correctActual, endFocus, fetchActualCorrections, fetchActuals, fetchCurrentFocus, pauseFocus, recordManualActual, resumeFocus, startFocus, type Actual } from "../api/focus";
import { fetchWorkItems } from "../api/work";
import { renderWithProviders } from "../test-utils";
import { FocusPage } from "./FocusPage";

vi.mock("../api/focus", () => ({ correctActual: vi.fn(), endFocus: vi.fn(), fetchActualCorrections: vi.fn().mockResolvedValue([]), fetchActuals: vi.fn().mockResolvedValue([]), fetchCurrentFocus: vi.fn(), pauseFocus: vi.fn(), recordManualActual: vi.fn(), resumeFocus: vi.fn(), startFocus: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn() }));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

const work = [{ id: "work-1", title: "Draft the launch story", description: "Keep it useful.", remainingMinutes: 45, sourceLabel: "Cadence" }];
const session = { id: "focus-1", workItemId: "work-1", title: "Draft the launch story", mode: "open", state: "running", startedAtUnixSeconds: 1000n, endedAtUnixSeconds: 0n, activeSeconds: 0n, wallSeconds: 0n, revision: 1n, activeStartedAtUnixSeconds: 1000n };

describe("FocusPage", () => {
  it("starts a real session for the selected work item", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchCurrentFocus).mockResolvedValue(null);
    vi.mocked(fetchActuals).mockResolvedValue([]);
    vi.mocked(fetchWorkItems).mockResolvedValue(work as never);
    vi.mocked(startFocus).mockResolvedValue(session as never);

    renderWithProviders(<FocusPage />);

    expect(await screen.findByRole("heading", { name: "Draft the launch story" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Work item"), { target: { value: "work-1" } });
    await user.click(screen.getByRole("button", { name: "Start focus" }));
    expect(startFocus).toHaveBeenCalledWith({ workItemId: "work-1", title: "Draft the launch story", mode: "open" });
  });

  it("exposes pause and resume as persisted transitions", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchCurrentFocus).mockResolvedValueOnce(session as never).mockResolvedValue({ ...session, state: "paused", revision: 2n } as never);
    vi.mocked(fetchActuals).mockResolvedValue([]);
    vi.mocked(fetchWorkItems).mockResolvedValue(work as never);
    vi.mocked(pauseFocus).mockResolvedValue({ ...session, state: "paused", revision: 2n } as never);
    vi.mocked(resumeFocus).mockResolvedValue({ ...session, revision: 3n } as never);

    renderWithProviders(<FocusPage />);
    await user.click(await screen.findByRole("button", { name: "Pause" }));
    expect(pauseFocus).toHaveBeenCalledWith(session);

    await user.click(await screen.findByRole("button", { name: "Resume" }));
    expect(resumeFocus).toHaveBeenCalledWith({ ...session, state: "paused", revision: 2n });
  });

  it("reports an honest transition failure", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchCurrentFocus).mockResolvedValue(session as never);
    vi.mocked(fetchActuals).mockResolvedValue([]);
    vi.mocked(fetchWorkItems).mockResolvedValue(work as never);
    vi.mocked(pauseFocus).mockRejectedValue(new Error("conflict"));

    renderWithProviders(<FocusPage />);
    await user.click(await screen.findByRole("button", { name: "Pause" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("That transition did not save");
    expect(endFocus).not.toHaveBeenCalled();
  });

  it("does not invent a session when the current read is unavailable", async () => {
    vi.mocked(fetchCurrentFocus).mockRejectedValue(new Error("offline"));
    vi.mocked(fetchActuals).mockResolvedValue([]);
    vi.mocked(fetchWorkItems).mockResolvedValue([]);
    renderWithProviders(<FocusPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("focus session is unavailable");
    expect(screen.queryByRole("button", { name: "Start focus" })).not.toBeInTheDocument();
  });

  it("records a manual actual and preserves an explicit correction path", async () => {
    const user = userEvent.setup();
    const actual: Actual = { id: "actual-1", workItemId: "", title: "Review notes", localDate: "2026-09-19", reportedMinutes: 45n, certainty: "user_reported_approximate", note: "", createdAtUnixSeconds: 0n, revision: 1n };
    vi.mocked(fetchCurrentFocus).mockResolvedValue(null);
    vi.mocked(fetchWorkItems).mockResolvedValue(work as never);
    vi.mocked(fetchActuals).mockResolvedValue([actual]);
    vi.mocked(fetchActualCorrections).mockResolvedValue([{ id: "correction-1", actualId: actual.id, previousMinutes: 45n, newMinutes: 30n, previousCertainty: "user_reported_approximate", newCertainty: "user_reported_approximate", reason: "Removed interruption", createdAtUnixSeconds: 0n }] as never);
    vi.mocked(recordManualActual).mockResolvedValue(actual);
    vi.mocked(correctActual).mockResolvedValue({ ...actual, reportedMinutes: 30n, revision: 2n });
    renderWithProviders(<FocusPage />);
    expect(await screen.findByText("1 correction preserved")).toBeInTheDocument();
    await user.type(await screen.findByLabelText("What did you work on?"), "Review notes");
    await user.type(screen.getByLabelText("Minutes"), "45");
    await user.click(screen.getByRole("button", { name: "Record actual" }));
    expect(recordManualActual).toHaveBeenCalledWith(expect.objectContaining({ title: "Review notes", reportedMinutes: 45 }));
    await user.click(await screen.findByRole("button", { name: "Correct" }));
    const correctionInput = screen.getAllByLabelText("Minutes")[1];
    if (!correctionInput) throw new Error("correction minutes input missing");
    await user.clear(correctionInput);
    await user.type(correctionInput, "30");
    await user.type(screen.getByLabelText("Why did it change?"), "Removed interruption");
    await user.click(screen.getByRole("button", { name: "Save correction" }));
    expect(correctActual).toHaveBeenCalledWith(actual, expect.objectContaining({ reportedMinutes: 30 }));
  });
});
