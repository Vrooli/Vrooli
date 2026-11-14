import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import type { ReindexStatus, ReindexTrigger } from "../../api/reindex";

vi.mock("../../api/reindex", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/reindex")>();
  let counter = 0;
  const reindex = vi.fn(
    (scenario: string, dryRun: boolean): Promise<ReindexTrigger> => {
      if (scenario === "broken") return Promise.reject(new Error("boom"));
      counter += 1;
      return Promise.resolve({
        jobId: `job-${counter}`,
        plannedUpserts: 3,
        plannedDeletes: 1,
        dryRun,
      });
    },
  );
  const reindexStatus = vi.fn(
    (jobId: string): Promise<ReindexStatus> =>
      Promise.resolve({
        jobId,
        state: "running",
        processed: 1,
        total: 3,
        error: "",
      }),
  );
  const reindexCancel = vi.fn();
  return { ...actual, reindex, reindexStatus, reindexCancel };
});

import { ReindexPage } from "./ReindexPage";
import { reindex } from "../../api/reindex";

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(reindex).mockClear();
});

describe("ReindexPage", () => {
  it("renders the trigger form and empty jobs state", () => {
    renderWithProviders(<ReindexPage />);
    expect(screen.getByTestId(selectors.reindex.form)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.reindex.emptyJobs)).toBeInTheDocument();
    expect(reindex).not.toHaveBeenCalled();
  });

  it("triggers reindex for a named scenario and adds a job row", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ReindexPage />);
    await user.type(screen.getByTestId(selectors.reindex.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.reindex.submit));
    await waitFor(() => expect(reindex).toHaveBeenCalledWith("ui-health", false));
    const list = await screen.findByTestId(selectors.reindex.jobsList);
    expect(within(list).getAllByRole("article")).toHaveLength(1);
  });

  it("requires confirmation when triggering reindex for all scenarios", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ReindexPage />);
    // Empty scenario name + submit → confirm modal opens
    await user.click(screen.getByTestId(selectors.reindex.submit));
    expect(screen.getByTestId(selectors.reindex.confirmModal)).toBeInTheDocument();
    expect(reindex).not.toHaveBeenCalled();
    // Cancel closes the modal without triggering
    await user.click(screen.getByTestId(selectors.reindex.confirmCancel));
    expect(reindex).not.toHaveBeenCalled();
    expect(screen.queryByTestId(selectors.reindex.confirmModal)).not.toBeInTheDocument();
    // Re-submit and accept
    await user.click(screen.getByTestId(selectors.reindex.submit));
    await user.click(screen.getByTestId(selectors.reindex.confirmAccept));
    await waitFor(() => expect(reindex).toHaveBeenCalledWith("", false));
  });

  it("rejects an invalid scenario name without calling the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ReindexPage />);
    await user.type(screen.getByTestId(selectors.reindex.scenarioInput), "Bad Name!");
    await user.click(screen.getByTestId(selectors.reindex.submit));
    expect(reindex).not.toHaveBeenCalled();
    expect(await screen.findByTestId(selectors.reindex.error)).toBeInTheDocument();
  });

  it("passes the dry-run flag through", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ReindexPage />);
    await user.type(screen.getByTestId(selectors.reindex.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.reindex.dryRunInput));
    await user.click(screen.getByTestId(selectors.reindex.submit));
    await waitFor(() => expect(reindex).toHaveBeenCalledWith("ui-health", true));
  });
});
