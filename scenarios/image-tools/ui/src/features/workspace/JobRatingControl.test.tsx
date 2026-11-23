/**
 * JobRatingControl tests — the operator's 1-5 rating of a finished result. The
 * rating client is a hand-built fake, so the star taps, the optional note, the
 * existing-rating load, removal and the error path are exercised without a
 * transport.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import type { JobRating } from "../../api/jobs";
import { JobRatingControl } from "./JobRatingControl";
import type { RatingClient } from "./ratingClient";

const rating = (score: number, note = ""): JobRating => ({ jobId: "job-1", score, note }) as JobRating;

const fakeClient = (overrides: Partial<RatingClient> = {}): RatingClient => ({
  load: vi.fn().mockResolvedValue(undefined),
  rate: vi.fn((_job: string, score: number, note: string) => Promise.resolve(rating(score, note))),
  clear: vi.fn().mockResolvedValue(undefined),
  ...overrides,
});

const score = (n: number) => screen.getByTestId(selectors.workspace.ratingScore({ score: n }));

describe("JobRatingControl", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("offers five touch-sized scores and saves one on a single tap", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);

    for (const n of [1, 2, 3, 4, 5]) {
      expect(score(n)).toHaveClass("h-11", "w-11");
      expect(score(n)).toHaveAttribute("aria-pressed", "false");
    }
    expect(screen.queryByTestId(selectors.workspace.rating.note)).not.toBeInTheDocument();

    await user.click(score(4));

    await waitFor(() => expect(screen.getByTestId(selectors.workspace.rating.saved)).toHaveTextContent("Rated 4 of 5"));
    expect(client.rate).toHaveBeenCalledWith("job-1", 4, "");
    expect(score(4)).toHaveAttribute("aria-pressed", "true");
    expect(score(5)).toHaveAttribute("aria-pressed", "false");
  });

  it("saves a note with the current score, and only when the note changed", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);
    await user.click(score(2));
    const note = await screen.findByTestId(selectors.workspace.rating.note);
    expect(screen.getByTestId(selectors.workspace.rating.save)).toBeDisabled();

    await user.type(note, "hair colour drifted");
    await user.click(screen.getByTestId(selectors.workspace.rating.save));

    await waitFor(() => expect(client.rate).toHaveBeenLastCalledWith("job-1", 2, "hair colour drifted"));
    await waitFor(() => expect(screen.getByTestId(selectors.workspace.rating.save)).toBeDisabled());
  });

  it("shows the rating a job already has", async () => {
    const client = fakeClient({ load: vi.fn().mockResolvedValue(rating(5, "kept the face")) });
    renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);

    await waitFor(() => expect(score(5)).toHaveAttribute("aria-pressed", "true"));
    expect(screen.getByTestId(selectors.workspace.rating.note)).toHaveValue("kept the face");
  });

  it("removes the rating", async () => {
    const client = fakeClient({ load: vi.fn().mockResolvedValue(rating(3)) });
    const user = userEvent.setup();
    renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);
    await user.click(await screen.findByTestId(selectors.workspace.rating.clear));

    await waitFor(() => expect(screen.queryByTestId(selectors.workspace.rating.saved)).not.toBeInTheDocument());
    expect(client.clear).toHaveBeenCalledWith("job-1");
    expect(score(3)).toHaveAttribute("aria-pressed", "false");
  });

  it("reports a failed save and keeps the control usable", async () => {
    const client = fakeClient({ rate: vi.fn().mockRejectedValue(new Error("boom")) });
    const user = userEvent.setup();
    renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);
    await user.click(score(1));

    expect(await screen.findByTestId(selectors.workspace.rating.error)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.workspace.rating.saved)).not.toBeInTheDocument();
    expect(score(1)).not.toBeDisabled();
  });

  it("starts clean when pointed at a different job", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    const { rerender } = renderWithProviders(<JobRatingControl jobId="job-1" client={client} />);
    await user.click(score(5));
    await screen.findByTestId(selectors.workspace.rating.saved);

    rerender(<JobRatingControl jobId="job-2" client={client} />);

    await waitFor(() => expect(screen.queryByTestId(selectors.workspace.rating.saved)).not.toBeInTheDocument());
    expect(client.load).toHaveBeenLastCalledWith("job-2");
  });
});
