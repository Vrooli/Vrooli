/**
 * SafetyFeedbackControl tests — the operator's judgement of a reviewer
 * refusal/flag. The client is a hand-built fake, so the one-tap labels, the
 * optional note, the existing-judgement load, removal and the error path are
 * exercised without a transport.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { SafetyFeedbackCaseSchema } from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { JobState } from "../../api/jobs";
import { SafetyFeedbackLabel } from "../../api/safetyFeedback";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { makeJob } from "../jobs/mocks/factories";
import { SafetyFeedbackControl } from "./SafetyFeedbackControl";
import type { SafetyFeedbackClient } from "../safety/safetyFeedbackClient";

const reviewed = (meta: Record<string, string> = {}, state = JobState.FAILED) =>
  makeJob({
    id: "job-1",
    operation: "text_to_image",
    state,
    resultMeta: {
      safety_review: "refused",
      safety_action: "refuse",
      safety_reason: "sexual_unassessable",
      safety_evidence: `{"action":"refuse","reason_code":"sexual_unassessable"}`,
      ...meta,
    },
  });

const stored = (label: SafetyFeedbackLabel, note = "") =>
  create(SafetyFeedbackCaseSchema, { id: 1n, jobId: "job-1", label, note, outputRef: "" });

const fakeClient = (overrides: Partial<SafetyFeedbackClient> = {}): SafetyFeedbackClient => ({
  load: vi.fn().mockResolvedValue(null),
  record: vi.fn((_job: string, choice) =>
    Promise.resolve(
      stored(
        choice === "legitimate"
          ? SafetyFeedbackLabel.FALSE_POSITIVE
          : choice === "confirmed"
            ? SafetyFeedbackLabel.CONFIRMED
            : SafetyFeedbackLabel.UNCERTAIN,
      ),
    ),
  ),
  clear: vi.fn().mockResolvedValue(true),
  ...overrides,
});

const button = (choice: "legitimate" | "confirmed" | "uncertain") =>
  screen.getByTestId(selectors.activity.safetyFeedback[choice]);

describe("SafetyFeedbackControl", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders nothing for a job the reviewer passed", () => {
    const job = makeJob({
      state: JobState.SUCCEEDED,
      resultMeta: { safety_review: "passed", safety_action: "allow", safety_reason: "clean" },
    });
    renderWithProviders(<SafetyFeedbackControl job={job} client={fakeClient()} />);
    expect(screen.queryByTestId(selectors.activity.safetyFeedback.root)).not.toBeInTheDocument();
  });

  it("labels a refusal legitimate in one tap", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    renderWithProviders(<SafetyFeedbackControl job={reviewed()} client={client} />);

    expect(screen.getByTestId(selectors.activity.safetyFeedback.root)).toBeInTheDocument();
    expect(screen.getByText(/sexual_unassessable/)).toBeInTheDocument();
    await user.click(button("legitimate"));

    await waitFor(() => expect(screen.getByTestId(selectors.activity.safetyFeedback.saved)).toHaveTextContent("legitimate"));
    expect(client.record).toHaveBeenCalledWith("job-1", "legitimate", "");
    expect(button("legitimate")).toHaveAttribute("aria-pressed", "true");
  });

  it("shows an existing judgement and saves a changed note with it", async () => {
    const client = fakeClient({
      load: vi.fn().mockResolvedValue(stored(SafetyFeedbackLabel.CONFIRMED, "correct refusal")),
    });
    const user = userEvent.setup();
    renderWithProviders(<SafetyFeedbackControl job={reviewed()} client={client} />);

    await waitFor(() => expect(button("confirmed")).toHaveAttribute("aria-pressed", "true"));
    const note = screen.getByTestId(selectors.activity.safetyFeedback.note);
    expect(note).toHaveValue("correct refusal");
    expect(screen.getByTestId(selectors.activity.safetyFeedback.save)).toBeDisabled();

    await user.clear(note);
    await user.type(note, "clearly a child");
    await user.click(screen.getByTestId(selectors.activity.safetyFeedback.save));

    await waitFor(() => expect(client.record).toHaveBeenCalledWith("job-1", "confirmed", "clearly a child"));
  });

  it("removes a judgement", async () => {
    const client = fakeClient({ load: vi.fn().mockResolvedValue(stored(SafetyFeedbackLabel.FALSE_POSITIVE)) });
    const user = userEvent.setup();
    renderWithProviders(<SafetyFeedbackControl job={reviewed()} client={client} />);
    await user.click(await screen.findByTestId(selectors.activity.safetyFeedback.remove));

    await waitFor(() => expect(screen.queryByTestId(selectors.activity.safetyFeedback.saved)).not.toBeInTheDocument());
    expect(client.clear).toHaveBeenCalledWith("job-1");
  });

  it("reports a failed save and keeps the control usable", async () => {
    const client = fakeClient({ record: vi.fn().mockRejectedValue(new Error("boom")) });
    const user = userEvent.setup();
    renderWithProviders(<SafetyFeedbackControl job={reviewed()} client={client} />);
    await user.click(button("uncertain"));

    expect(await screen.findByTestId(selectors.activity.safetyFeedback.error)).toBeInTheDocument();
    expect(button("uncertain")).not.toBeDisabled();
  });

  it("shows a flagged succeeded result too", () => {
    renderWithProviders(
      <SafetyFeedbackControl
        job={reviewed(
          { safety_review: "flagged", safety_action: "flag", safety_reason: "sexual_adult" },
          JobState.SUCCEEDED,
        )}
        client={fakeClient()}
      />,
    );
    expect(screen.getByTestId(selectors.activity.safetyFeedback.root)).toBeInTheDocument();
  });
});
