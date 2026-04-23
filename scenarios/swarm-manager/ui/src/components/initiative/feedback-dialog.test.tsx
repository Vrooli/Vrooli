import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FeedbackDialog } from "./feedback-dialog";
import { FeedbackLockConflictError } from "../../services/feedback-service";
import { selectors } from "../../consts/selectors";
import type { FeedbackRound } from "../../types";

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
});
