import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FeedbackDialog } from "./feedback-dialog";
import type { FeedbackDialogItem } from "./feedback-dialog";
import { buildEnvelope, pruneActionsForSelection } from "./feedback-dialog-envelope";
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

  // Help block toggles regardless of items prop; default closed.
  it("renders the help block collapsed by default and toggles open", async () => {
    renderDialog();
    const block = screen.getByTestId(selectors.feedback.dialogHelpBlock);
    // Initially body absent (only header rendered).
    expect(within(block).queryByText(/proposes a checklist/i)).toBeNull();
    await userEvent.click(within(block).getByTestId(selectors.feedback.dialogHelpBlockToggle));
    expect(within(block).getByText(/proposes a checklist/i)).toBeInTheDocument();
  });

  // When items prop is omitted, the picker and quick actions stay
  // hidden — back-compat with callers that haven't wired items yet.
  it("hides the picker but keeps quick actions when items prop is absent", () => {
    // Gap-and-drift sweeps (identify_missing_work / reconcile_with_code_drift)
    // are the most common feedback use case on completed initiatives where
    // there are no live items to select. The actions row must stay visible
    // so those lenses are reachable; the picker can still hide because
    // there's nothing to pick.
    renderDialog();
    expect(screen.queryByTestId(selectors.feedback.dialogTargetPicker)).toBeNull();
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionIdentifyMissing)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionReconcile)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionReframe)).toBeInTheDocument();
    // Split and merge are rendered but disabled (no items to operate on).
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionSplit)).toBeDisabled();
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionMerge)).toBeDisabled();
  });
});

// ---------------------------------------------------------------------------
// Quick Actions surface (Plan A)
// ---------------------------------------------------------------------------

const SAMPLE_ITEMS: FeedbackDialogItem[] = [
  { ref: "execute/alpha", title: "Alpha" },
  { ref: "execute/beta", title: "Beta" },
  { ref: "execute/gamma", title: "Gamma" },
];

function renderWithItems(props?: Partial<React.ComponentProps<typeof FeedbackDialog>>) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <FeedbackDialog
        initiativeName="my-initiative"
        isOpen
        onClose={vi.fn()}
        onSubmitted={vi.fn()}
        items={SAMPLE_ITEMS}
        {...props}
      />
    </QueryClientProvider>,
  );
}

describe("FeedbackDialog · Quick Actions", () => {
  beforeEach(() => {
    mockStart.mockReset();
    mockLockStatus.mockReset();
    mockLockStatus.mockResolvedValue({ locked: false });
    try {
      localStorage.clear();
    } catch {
      /* ignore */
    }
  });

  it("renders the picker collapsed and the five quick actions", () => {
    renderWithItems();
    expect(screen.getByTestId(selectors.feedback.dialogTargetPicker)).toBeInTheDocument();
    // Picker collapsed → list rows aren't rendered yet.
    expect(screen.queryAllByTestId(selectors.feedback.dialogTargetPickerItem)).toHaveLength(0);
    // All 5 quick actions are present.
    for (const id of [
      selectors.feedback.dialogQuickActionSplit,
      selectors.feedback.dialogQuickActionMerge,
      selectors.feedback.dialogQuickActionIdentifyMissing,
      selectors.feedback.dialogQuickActionReconcile,
      selectors.feedback.dialogQuickActionReframe,
    ]) {
      expect(screen.getByTestId(id)).toBeInTheDocument();
    }
  });

  it("gates split / merge based on selection count and mutual exclusion", async () => {
    renderWithItems();
    const splitBtn = screen.getByTestId(selectors.feedback.dialogQuickActionSplit);
    const mergeBtn = screen.getByTestId(selectors.feedback.dialogQuickActionMerge);

    // Initially zero selected → both disabled.
    expect(splitBtn).toBeDisabled();
    expect(mergeBtn).toBeDisabled();

    // Open picker, pick 1 item: split enabled, merge still disabled (<2).
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerToggle));
    const rows = await screen.findAllByTestId(selectors.feedback.dialogTargetPickerItem);
    await userEvent.click(rows[0]!);
    expect(splitBtn).toBeEnabled();
    expect(mergeBtn).toBeDisabled();

    // Pick a second: merge enabled too.
    await userEvent.click(rows[1]!);
    expect(mergeBtn).toBeEnabled();

    // Selecting Split should disable Merge (mutual exclusion) — and the
    // converse: clicking Merge after Split is selected first switches
    // them. Verify split is selected first.
    await userEvent.click(splitBtn);
    expect(splitBtn).toHaveAttribute("aria-pressed", "true");
    // Merge is now disabled because split is active.
    expect(mergeBtn).toBeDisabled();
  });

  it("treats reframe_scope as solo: selects clear other quick actions and vice versa", async () => {
    renderWithItems();
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerToggle));
    const rows = await screen.findAllByTestId(selectors.feedback.dialogTargetPickerItem);
    await userEvent.click(rows[0]!);
    await userEvent.click(rows[1]!);

    const identify = screen.getByTestId(selectors.feedback.dialogQuickActionIdentifyMissing);
    const reframe = screen.getByTestId(selectors.feedback.dialogQuickActionReframe);

    // Pick identify-missing first.
    await userEvent.click(identify);
    expect(identify).toHaveAttribute("aria-pressed", "true");

    // Now picking reframe should clear identify.
    await userEvent.click(reframe);
    expect(reframe).toHaveAttribute("aria-pressed", "true");
    expect(identify).toHaveAttribute("aria-pressed", "false");

    // And selecting any other quick action clears reframe.
    await userEvent.click(identify);
    expect(reframe).toHaveAttribute("aria-pressed", "false");
    expect(identify).toHaveAttribute("aria-pressed", "true");
  });

  it("emits raw text when no actions and no items are selected", async () => {
    let resolvePending: (round: FeedbackRound) => void = () => {};
    mockStart.mockImplementationOnce(
      () =>
        new Promise<FeedbackRound>((resolve) => {
          resolvePending = resolve;
        }),
    );
    renderWithItems();
    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "free prose only");
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    await waitFor(() => expect(mockStart).toHaveBeenCalled());
    const callArgs = mockStart.mock.calls[0]![1];
    expect(callArgs.text).toBe("free prose only");
    expect(callArgs.text).not.toContain("<selection");
    expect(callArgs.text).not.toContain("<requested_actions");
    resolvePending(makeRound());
  });

  it("wraps in XML envelope when items or actions are selected", async () => {
    let resolvePending: (round: FeedbackRound) => void = () => {};
    mockStart.mockImplementationOnce(
      () =>
        new Promise<FeedbackRound>((resolve) => {
          resolvePending = resolve;
        }),
    );
    renderWithItems();
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerToggle));
    const rows = await screen.findAllByTestId(selectors.feedback.dialogTargetPickerItem);
    await userEvent.click(rows[0]!); // execute/alpha
    await userEvent.click(rows[1]!); // execute/beta
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogQuickActionMerge));
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogQuickActionIdentifyMissing));
    await userEvent.type(screen.getByTestId(selectors.feedback.dialogText), "and add tests");
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogSubmit));

    await waitFor(() => expect(mockStart).toHaveBeenCalled());
    const text = mockStart.mock.calls[0]![1].text;
    expect(text).toContain('<item ref="execute/alpha" />');
    expect(text).toContain('<item ref="execute/beta" />');
    expect(text).toContain('<action name="merge_coupled" />');
    expect(text).toContain('<action name="identify_missing_work" />');
    expect(text).toContain("<user_note>");
    expect(text).toContain("and add tests");
    resolvePending(makeRound());
  });

  it("hides quick actions and disables picker when round type is Note", async () => {
    renderWithItems();
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTypeNote));
    // Quick actions hidden under note type.
    expect(screen.queryByTestId(selectors.feedback.dialogQuickActionSplit)).toBeNull();
    expect(screen.queryByTestId(selectors.feedback.dialogQuickActionMerge)).toBeNull();
    // Picker also hidden.
    expect(screen.queryByTestId(selectors.feedback.dialogTargetPicker)).toBeNull();
    // Help block remains visible (informational under all types).
    expect(screen.getByTestId(selectors.feedback.dialogHelpBlock)).toBeInTheDocument();
  });

  it("Select all / Select none update the picker selection summary", async () => {
    renderWithItems();
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerToggle));
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerSelectAll));

    const rows = await screen.findAllByTestId(selectors.feedback.dialogTargetPickerItem);
    rows.forEach((row) => {
      const cb = within(row).getByRole("checkbox");
      expect(cb).toBeChecked();
    });

    await userEvent.click(screen.getByTestId(selectors.feedback.dialogTargetPickerSelectNone));
    rows.forEach((row) => {
      const cb = within(row).getByRole("checkbox");
      expect(cb).not.toBeChecked();
    });
  });

  it("allows submit when ≥1 quick action is selected even with empty text", async () => {
    let resolvePending: (round: FeedbackRound) => void = () => {};
    mockStart.mockImplementationOnce(
      () =>
        new Promise<FeedbackRound>((resolve) => {
          resolvePending = resolve;
        }),
    );
    renderWithItems();
    // Identify-missing is gateable with zero items selected — the user
    // can ask the agent to investigate the whole initiative.
    await userEvent.click(screen.getByTestId(selectors.feedback.dialogQuickActionIdentifyMissing));
    const submit = screen.getByTestId(selectors.feedback.dialogSubmit);
    expect(submit).toBeEnabled();
    await userEvent.click(submit);
    await waitFor(() => expect(mockStart).toHaveBeenCalled());
    resolvePending(makeRound());
  });
});

// ---------------------------------------------------------------------------
// Pure helpers — envelope assembly + action pruning
// ---------------------------------------------------------------------------

describe("buildEnvelope (Plan A)", () => {
  it("assembles selection + requested_actions + user_note", () => {
    const env = buildEnvelope({
      items: ["execute/alpha", "execute/beta"],
      actions: ["merge_coupled", "identify_missing_work"],
      note: "  hello world  ",
    });
    expect(env).toContain('<item ref="execute/alpha" />');
    expect(env).toContain('<item ref="execute/beta" />');
    expect(env).toContain('<action name="merge_coupled" />');
    expect(env).toContain('<action name="identify_missing_work" />');
    expect(env).toContain("hello world");
  });

  it("emits empty selection / actions blocks when corresponding inputs are empty", () => {
    const env = buildEnvelope({ items: [], actions: [], note: "x" });
    expect(env).toContain("<selection></selection>");
    expect(env).toContain("<requested_actions></requested_actions>");
  });

  it("escapes XML-special characters in refs", () => {
    const env = buildEnvelope({
      items: ['execute/with"quote', "execute/<lt"],
      actions: [],
      note: "",
    });
    expect(env).toContain("&quot;");
    expect(env).toContain("&lt;");
  });

  it("prunes split/merge from action set when selection size drops below threshold", () => {
    const startSet = new Set(["split_oversized" as const, "merge_coupled" as const]);
    expect(pruneActionsForSelection(startSet, 0)).toEqual(new Set());
    const startSet2 = new Set(["split_oversized" as const, "merge_coupled" as const]);
    // Merge requires ≥2; with 1 selected, merge drops but split remains.
    expect(pruneActionsForSelection(startSet2, 1)).toEqual(new Set(["split_oversized"]));
  });
});
