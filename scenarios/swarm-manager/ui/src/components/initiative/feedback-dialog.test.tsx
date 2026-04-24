import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FeedbackDialog } from "./feedback-dialog";
import { FeedbackBusyError, FeedbackLockConflictError } from "../../services/feedback-service";
import { selectors } from "../../consts/selectors";
import type { FeedbackRound, LockStatusResponse } from "../../types";

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

vi.mock("../../services/feedback-service", async () => {
  const actual = await vi.importActual<typeof import("../../services/feedback-service")>(
    "../../services/feedback-service",
  );
  return {
    ...actual,
    feedbackService: {
      start: vi.fn(),
      // Default to "nothing is running" so existing tests don't have to
      // opt in. Lock-preflight tests override this per-case.
      lockStatus: vi.fn(async () => ({ locked: false }) as LockStatusResponse),
    },
  };
});

// IndexedDB isn't available in jsdom — the attachment hook falls back to an
// in-memory shim, but the open() call can still reject. Mock useIndexedDBAttachments
// outright so the component renders deterministically.
vi.mock("../../hooks/useIndexedDBAttachments", () => ({
  useIndexedDBAttachments: () => ({
    attachments: [],
    addFile: vi.fn(),
    removeFile: vi.fn(),
    clearAll: vi.fn(),
    getFiles: () => [],
  }),
}));

const { feedbackService } = await import("../../services/feedback-service");
const mockStart = vi.mocked(feedbackService.start);
const mockLockStatus = vi.mocked(feedbackService.lockStatus);

function renderDialog(props?: Partial<React.ComponentProps<typeof FeedbackDialog>>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <FeedbackDialog
        initiativeName="my-initiative"
        isOpen
        onClose={vi.fn()}
        onSubmitted={vi.fn()}
        {...props}
      />
    </QueryClientProvider>,
  );
}

function makeRound(overrides: Partial<FeedbackRound> = {}): FeedbackRound {
  return {
    initiative_name: "my-initiative",
    number: 1,
    slug: "hello",
    type: "feedback",
    status: "agent_thinking",
    submission: { text: "hello", created_at: "2026-04-23T00:00:00Z" },
    thread: [],
    proposals: [],
    created_at: "2026-04-23T00:00:00Z",
    updated_at: "2026-04-23T00:00:00Z",
    ...overrides,
  };
}

beforeEach(() => {
  mockStart.mockReset();
  mockLockStatus.mockReset();
  mockLockStatus.mockResolvedValue({ locked: false });
  try {
    localStorage.clear();
  } catch {
    /* jsdom localStorage may not be available in some configs */
  }
});
afterEach(() => cleanup());

describe("FeedbackDialog", () => {
  it("renders the three type chips with research disabled", () => {
    renderDialog();
    expect(screen.getByTestId(selectors.feedback.dialogTypeFeedback)).toBeEnabled();
    expect(screen.getByTestId(selectors.feedback.dialogTypeNote)).toBeEnabled();
    expect(screen.getByTestId(selectors.feedback.dialogTypeResearch)).toBeDisabled();
  });

  it("disables submit until text is entered", async () => {
    renderDialog();
    const submit = screen.getByTestId(selectors.feedback.dialogSubmit);
    expect(submit).toBeDisabled();

    const textarea = screen.getByTestId(selectors.feedback.dialogText);
    await userEvent.type(textarea, "something");
    expect(submit).toBeEnabled();
  });

  it("submits with the selected round type and calls onSubmitted on success", async () => {
    mockStart.mockResolvedValue(makeRound({ type: "note", status: "dismissed" }));
    const onSubmitted = vi.fn();
    const onClose = vi.fn();
    renderDialog({ onSubmitted, onClose });

    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTypeNote));
    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "quick note");
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    await waitFor(() => expect(mockStart).toHaveBeenCalled());
    expect(mockStart).toHaveBeenCalledWith(
      "my-initiative",
      expect.objectContaining({ type: "note", text: "quick note" }),
    );
    expect(onSubmitted).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("keeps submit disabled while a prior submit is in flight", async () => {
    // Hold the submit promise open so the mutation stays pending. During
    // that window a second click must not fire another `start` call — the
    // dialog's `canSubmit` gate is our only line of defense against a user
    // double-clicking and accidentally creating two rounds.
    let resolvePending: (round: FeedbackRound) => void = () => {};
    mockStart.mockImplementationOnce(
      () =>
        new Promise<FeedbackRound>((resolve) => {
          resolvePending = resolve;
        }),
    );
    renderDialog();

    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "first");
    const submit = screen.getByTestId(selectors.feedback.dialogSubmit);
    await userEvent.click(submit);

    // Pending: submit is disabled and clicking again is a no-op.
    await waitFor(() => expect(submit).toBeDisabled());
    await userEvent.click(submit);
    expect(mockStart).toHaveBeenCalledTimes(1);

    // Release the pending submit so React Testing Library doesn't leak the promise.
    resolvePending(makeRound());
    await waitFor(() => expect(mockStart).toHaveBeenCalledTimes(1));
  });

  it("surfaces a lock conflict and requires explicit override before resubmitting", async () => {
    mockStart.mockRejectedValueOnce(
      new FeedbackLockConflictError("locked", { run_id: "r1", purpose: "feedback" }),
    );
    renderDialog();

    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "attempt");
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    // Warning with override checkbox appears; submit stays disabled.
    const checkbox = await screen.findByTestId(selectors.feedback.dialogOverrideConfirm);
    expect(checkbox).toBeInTheDocument();
    expect(screen.getByTestId(selectors.feedback.dialogSubmit)).toBeDisabled();

    mockStart.mockResolvedValueOnce(makeRound());
    await userEvent.click(checkbox);
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    await waitFor(() => expect(mockStart).toHaveBeenCalledTimes(2));
    const secondCallArgs = mockStart.mock.calls[1]![1];
    expect(secondCallArgs.override).toBe(true);
  });

  // Proactive preflight: if the initiative is already locked when the
  // dialog opens, the warning renders immediately and the textarea is
  // disabled until the user ticks override. This is the "don't let me
  // type during an active agent" behavior the plan promised — the old
  // code only surfaced the warning after a 409 round-trip.
  it("renders the warning immediately when preflight says the initiative is locked", async () => {
    mockLockStatus.mockResolvedValue({
      locked: true,
      holder: { run_id: "active-run", purpose: "feedback", round_number: 3 },
    });
    renderDialog();

    const notice = await screen.findByTestId(selectors.feedback.dialogBlockerNotice);
    expect(notice).toBeInTheDocument();
    expect(notice.textContent).toContain("active-run");

    const textarea = screen.getByTestId(selectors.feedback.dialogText);
    expect(textarea).toBeDisabled();
    expect(screen.getByTestId(selectors.feedback.dialogSubmit)).toBeDisabled();
  });

  // When preflight reports busy backlog items, the dialog lists them by
  // ref so the user sees exactly which agents will be preempted before
  // committing to override. Distinguishable from a plain lock conflict:
  // here the holder is absent and activities carries the detail.
  it("renders a per-item busy notice when preflight returns item activities", async () => {
    mockLockStatus.mockResolvedValue({
      locked: false,
      item_activities: [
        { ref: "execute/foo", purpose: "execute", run_id: "run-foo" },
        { ref: "research/bar", purpose: "workshop", run_id: "run-bar" },
      ],
    });
    renderDialog();

    const notice = await screen.findByTestId(selectors.feedback.dialogBlockerNotice);
    expect(notice.textContent).toContain("execute/foo");
    expect(notice.textContent).toContain("research/bar");
    expect(screen.getByTestId(selectors.feedback.dialogText)).toBeDisabled();
  });

  // Note-type rounds don't spawn an agent, so the active-agent guard
  // doesn't apply. Switching to note while something is running must
  // unblock typing so the user can still log observations into the
  // feedback history. (Meta-optimizer signal capture shouldn't be
  // gated on whether an unrelated agent happens to be running.)
  it("does not block the note type even when the initiative is locked", async () => {
    mockLockStatus.mockResolvedValue({
      locked: true,
      holder: { run_id: "active", purpose: "feedback" },
    });
    renderDialog();

    // Initial render: feedback type is selected, notice present, textarea disabled.
    await screen.findByTestId(selectors.feedback.dialogBlockerNotice);
    expect(screen.getByTestId(selectors.feedback.dialogText)).toBeDisabled();

    // Switch to note — the notice goes away and typing is allowed.
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTypeNote));

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.feedback.dialogBlockerNotice)).toBeNull(),
    );
    expect(screen.getByTestId(selectors.feedback.dialogText)).toBeEnabled();
  });

  // A 409 with `activities` should land in FeedbackBusyError, not the
  // generic lock-conflict path. The dialog renders the same per-item
  // notice as the preflight busy case so the post-submit and pre-submit
  // experiences stay consistent.
  it("distinguishes a busy-error 409 from a lock-conflict 409", async () => {
    mockStart.mockRejectedValueOnce(
      new FeedbackBusyError("busy", [
        { ref: "execute/foo", purpose: "execute", run_id: "run-foo" },
      ]),
    );
    renderDialog();

    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "attempt");
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    const notice = await screen.findByTestId(selectors.feedback.dialogBlockerNotice);
    expect(notice.textContent).toContain("execute/foo");
    // No holder line — we're specifically in the busy-items path.
    expect(notice.textContent).not.toContain("holds the lock");
  });
});
