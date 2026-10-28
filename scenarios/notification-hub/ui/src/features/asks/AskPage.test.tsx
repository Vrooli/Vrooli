import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { conversationsClient, identityClient } from "../../api/notifications";
import { orderedOptions, remainingUntil, safeContextUrl } from "./askView";
import {
  clearAskDrafts,
  readAskDraft,
  readAskAttempt,
} from "./askDrafts";
import { TestAppRouter } from "../../app/routes";

vi.mock("../../api/notifications", () => ({
  identityClient: {
    getSession: vi.fn(() =>
      Promise.resolve({
        signedIn: true,
        subject: "owner-1",
        email: "owner@example.test",
      }),
    ),
  },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
}));

const inThreeHours = new Date(
  Date.now() + 3 * 60 * 60 * 1000 + 30_000,
).toISOString();

function pendingAsk() {
  return {
    id: "ask-1",
    question: "Close the BAS goal or re-aim at quality?",
    options: [
      { key: "close", label: "Close the goal" },
      { key: "re-aim", label: "Re-aim at quality" },
    ],
    recommended: "re-aim",
    recommendationReason: "Quality gaps remain.",
    defaultAnswer: "re-aim",
    reversible: true,
    defaultEligibleAt: inThreeHours,
    state: "pending",
    contextUrl: "",
    bigPicture: "",
    technicalDetails: "",
  };
}

function renderAsk() {
  return renderWithProviders(<TestAppRouter initialEntries={["/asks/ask-1"]} />, { withoutRouter: true });
}

describe("AskPage", () => {
  // The countdown and status copy are the behaviour under test, so read them
  // in English rather than as cimode keys.
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(identityClient.getSession).mockResolvedValue({
      signedIn: true,
      subject: "owner-1",
      email: "owner@example.test",
    } as never);
  });
  afterEach(() => {
    vi.clearAllMocks();
    clearAskDrafts();
  });

  it("allows recipient-scoped ask reads without login but requires a session to answer", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(conversationsClient.getAsk).toHaveBeenCalledWith({ askId: "ask-1" });
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
  });

  it("does not retry a recipient-scoped NotFound response or disclose whether the ask exists", async () => {
    vi.mocked(conversationsClient.getAsk).mockRejectedValue(
      new ConnectError("not found", Code.NotFound),
    );
    renderAsk();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This decision was not found, or it is not addressed to you.",
    );
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(1);
  });

  it("keeps ask details readable and hides answer controls immediately when the session expires", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    renderAsk();
    const options = await screen.findAllByTestId(selectors.asks.option);
    await user.click(options[1]!);
    await user.type(screen.getByTestId(selectors.asks.note), "keep this draft");

    window.dispatchEvent(new Event("notification-hub:session-expired"));

    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "keep this draft" });
  });

  it("restores the subject-scoped unsent draft after a page reload", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    const first = renderAsk();
    const options = await screen.findAllByTestId(selectors.asks.option);
    await user.click(options.find((option) => option.getAttribute("data-option-key") === "close")!);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "resume after refresh");
    first.unmount();

    renderAsk();
    const restored = await screen.findAllByTestId(selectors.asks.option);
    expect(restored.find((option) => option.getAttribute("data-option-key") === "close")).toBeChecked();
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("resume after refresh");
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("keeps each owner's draft private when the verified session identity changes", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    const view = renderAsk();
    const options = await screen.findAllByTestId(selectors.asks.option);
    await user.click(options.find((option) => option.getAttribute("data-option-key") === "close")!);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "owner one's private note");

    view.queryClient.setQueryData(["owner-session"], {
      signedIn: true,
      subject: "owner-2",
      email: "other@example.test",
    });
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    const secondOwnerOptions = await screen.findAllByTestId(selectors.asks.option);
    await waitFor(() => expect(secondOwnerOptions.every((option) => !(option as HTMLInputElement).checked)).toBe(true));
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("");
    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "owner one's private note" });
    expect(readAskDraft("owner-2", "ask-1")).toEqual({ answer: "", note: "" });
  });

  it("shows the recommended option first and the default with its countdown", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    renderAsk();

    const options = await screen.findAllByTestId(selectors.asks.option);
    expect(
      options.map((option) => option.getAttribute("data-option-key")),
    ).toEqual(["re-aim", "close"]);
    expect(options.every((option) => !(option as HTMLInputElement).checked)).toBe(true);
    expect(within(options[0]!.parentElement!).getByText(/Recommended/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.defaultNotice)).toHaveTextContent(
      "Re-aim at quality applies in 3 h",
    );
  });

  it("keeps the selected answer form accessible by role and label", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    const view = renderAsk();

    expect(await screen.findAllByRole("radio")).toHaveLength(2);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(pendingAsk().question);
    expect(screen.getByRole("button", { name: "Submit answer" })).toBeDisabled();
    await expectNoA11yViolations(view.container);
  });

  it("shows a producer-withdrawn request without answer controls or answer attribution", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        ...pendingAsk(),
        state: "withdrawn",
        answer: "",
        answeredBy: "",
      },
    } as never);
    renderAsk();

    expect(
      await screen.findByText("The requester withdrew this decision; no answer was recorded."),
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(screen.queryByRole("button", { name: /answer/i })).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/note/i)).not.toBeInTheDocument();
  });

  it("shows an expired decision as read-only", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: { ...pendingAsk(), state: "expired" },
    } as never);
    renderAsk();

    expect(await screen.findByText("This decision expired without an answer.")).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(screen.queryByRole("button", { name: "Submit answer" })).not.toBeInTheDocument();
  });

  it("keeps the server deadline wording when a default is eligible only after the minimum delay", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: { ...pendingAsk(), defaultEligibleAt: "2026-10-19T10:00:00Z" },
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.defaultNotice)).toHaveTextContent(
      "If you do not answer, Re-aim at quality applies after the deadline, at least 12 hours after this reached you.",
    );
  });

  it("shows the purpose and technical context as separate disclosures", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        ...pendingAsk(),
        bigPicture: "This keeps the quality goal useful.",
        technicalDetails: "Two required platform checks still fail.",
      },
    } as never);
    renderAsk();

    expect(await screen.findByText("This keeps the quality goal useful.")).toBeInTheDocument();
    expect(screen.getByText("Two required platform checks still fail.")).toBeInTheDocument();
    expect(screen.getByText("Add a note (optional)")).toBeInTheDocument();
  });

  it("preserves the answer note while both context disclosures open and close", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        ...pendingAsk(),
        bigPicture: "This keeps the quality goal useful.",
        technicalDetails: "Two required platform checks still fail.",
      },
    } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    const note = await screen.findByTestId(selectors.asks.note);
    await user.type(note, "keep the macOS follow-up");
    const disclosures = screen.getAllByText(/Why this matters|Technical details/);
    expect(disclosures).toHaveLength(2);
    for (const label of disclosures) {
      const details = label.closest("details");
      expect(details).not.toBeNull();
      await user.click(label);
      expect(details).toHaveProperty("open", true);
      expect(note).toHaveValue("keep the macOS follow-up");
      expect(conversationsClient.answer).not.toHaveBeenCalled();
      await user.click(label);
      expect(details).toHaveProperty("open", false);
      expect(note).toHaveValue("keep the macOS follow-up");
      expect(conversationsClient.answer).not.toHaveBeenCalled();
    }

    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    );
    await user.click(close!);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    expect(conversationsClient.answer).toHaveBeenCalledWith({
      askId: "ask-1",
      answer: "close",
      note: "keep the macOS follow-up",
    });
  });

  it("requires a deliberate selection and submit, then confirms the answer from authoritative readback", async () => {
    let finishReadback!: (response: unknown) => void;
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: pendingAsk() } as never)
      .mockImplementationOnce(() => new Promise((resolve) => { finishReadback = resolve; }) as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await screen.findAllByTestId(selectors.asks.option);
    await userEvent.type(
      screen.getByTestId(selectors.asks.note),
      "keep macOS last",
    );
    const [, close] = await screen.findAllByTestId(selectors.asks.option);
    await userEvent.click(close!);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Submit answer" }));

    await waitFor(() =>
      expect(conversationsClient.answer).toHaveBeenCalledTimes(1),
    );
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    expect(screen.queryByText("Answer recorded: Close the goal")).not.toBeInTheDocument();
    expect(screen.getAllByText("Checking status…")).toHaveLength(2);
    expect(conversationsClient.answer).toHaveBeenCalledWith({
      askId: "ask-1",
      answer: "close",
      note: "keep macOS last",
    });
    finishReadback({
      ask: {
        ...pendingAsk(),
        state: "answered",
        answer: "close",
        answerLabel: "Close the goal",
        note: "keep macOS last",
      },
    });
    expect(await screen.findByText("Answer recorded: Close the goal")).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("preserves the immutable attempted answer and never replays after an uncertain response", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();

    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    const note = screen.getByTestId(selectors.asks.note);
    await user.type(note, "keep this attempt");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();
    expect(note).toHaveValue("keep this attempt");
    expect(screen.queryByRole("button", { name: "Submit answer" })).not.toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Check answer status" }));
    expect(await screen.findByText(/earlier answer may still finish/)).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2);
  });

  it("confirms an uncertain answer from recipient readback without replaying the mutation", async () => {
    const user = userEvent.setup();
    const answered = {
      ...pendingAsk(),
      state: "answered",
      answer: "close",
      note: "keep this attempt",
      answerLabel: "Close the goal",
    };
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: pendingAsk() } as never)
      .mockResolvedValueOnce({ ask: answered } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();

    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    await user.type(screen.getByTestId(selectors.asks.note), "keep this attempt");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Check answer status" }));
    expect(await screen.findByText("Answer recorded: Close the goal")).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2);
  });

  it("keeps status readback available after session expiry without restoring answer controls", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();
    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "private draft note");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();

    window.dispatchEvent(new Event("notification-hub:session-expired"));
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check answer status" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit answer" })).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.asks.option)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.asks.note)).not.toBeInTheDocument();
    expect(screen.queryByText("private draft note")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check answer status" }));
    expect(await screen.findByText(/earlier answer may still finish/)).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("restores an uncertain immutable attempt after refresh without replaying it", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    const first = renderAsk();
    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "submitted once");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();
    expect(readAskAttempt("owner-1", "ask-1")).toEqual({
      answer: "close",
      note: "submitted once",
      subject: "owner-1",
    });
    first.unmount();

    renderAsk();
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("submitted once");
    expect(screen.getByRole("button", { name: "Check answer status" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit answer" })).not.toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("keeps an uncertain attempt private and unreplayed after the verified identity changes", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: pendingAsk() } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    const view = renderAsk();
    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "owner one's private attempt");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    expect(await screen.findByText(/We could not confirm whether this answer was recorded/)).toBeInTheDocument();

    view.queryClient.setQueryData(["owner-session"], {
      signedIn: true,
      subject: "owner-2",
      email: "other@example.test",
    });
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    expect(await screen.findAllByTestId(selectors.asks.option)).toHaveLength(2);
    expect(screen.queryByText(/We could not confirm whether this answer was recorded/)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("");
    expect(readAskAttempt("owner-1", "ask-1")).toEqual({
      answer: "close",
      note: "owner one's private attempt",
      subject: "owner-1",
    });
    expect(readAskAttempt("owner-2", "ask-1")).toBeNull();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("keeps one pending attempt locked through an open read and confirms a delayed commit", async () => {
    const user = userEvent.setup();
    let finishAnswer!: (value: unknown) => void;
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: pendingAsk() } as never)
      .mockResolvedValueOnce({ ask: pendingAsk() } as never)
      .mockResolvedValueOnce({
        ask: { ...pendingAsk(), state: "answered", answer: "close", note: "delayed", answerLabel: "Close the goal" },
      } as never);
    vi.mocked(conversationsClient.answer).mockImplementation(
      () => new Promise((resolve) => { finishAnswer = resolve; }) as never,
    );
    renderAsk();
    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    await user.click(screen.getByText("Add a note (optional)"));
    await user.type(screen.getByTestId(selectors.asks.note), "delayed");
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));

    await user.click(screen.getByRole("button", { name: "Check answer status" }));
    expect(await screen.findByText(/earlier answer may still finish/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit answer" })).not.toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);

    finishAnswer({});
    expect(await screen.findByText("Answer recorded: Close the goal")).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("shows an answered decision without answer buttons", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        ...pendingAsk(),
        state: "answered",
        answer: "close",
        answerLabel: "Close the goal",
      },
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.status)).toHaveTextContent(
      "Answered: Close the goal",
    );
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
  });

  it("allows one deliberate later human answer after a default is applied", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: { ...pendingAsk(), state: "defaulted", answer: "re-aim", defaultAnswer: "re-aim", answerLabel: "Re-aim at quality" },
    } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({ answer: "close" } as never);
    renderAsk();

    const close = (await screen.findAllByTestId(selectors.asks.option)).find(
      (option) => option.getAttribute("data-option-key") === "close",
    )!;
    await user.click(close);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Submit answer" }));
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "close", note: "" });
  });

  it("keeps the original default and its time visible after a later answer", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: {
        ...pendingAsk(),
        state: "answered",
        answer: "close",
        answerLabel: "Close the goal",
        defaultAnswer: "re-aim",
        defaultAppliedAt: "2026-10-24T14:10:00Z",
        late: true,
      },
    } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    expect(status).toHaveTextContent("Recorded default");
    expect(status).toHaveTextContent("Re-aim at quality");
    expect(status).toHaveTextContent("Default applied");
    expect(status).toHaveTextContent("The requester may already have acted on the default.");
    expect(status.querySelector("time")).toHaveAttribute("dateTime", "2026-10-24T14:10:00Z");
  });
});

describe("askView", () => {
  it("orders the recommended option first and formats the remaining time", () => {
    expect(
      orderedOptions({
        options: pendingAsk().options,
        recommended: "re-aim",
      } as never).map((o) => o.key),
    ).toEqual(["re-aim", "close"]);
    expect(
      remainingUntil(
        "2026-10-07T12:30:00Z",
        Date.parse("2026-10-07T10:00:00Z"),
      ),
    ).toBe("2 h 30 min");
    expect(
      remainingUntil(
        "2026-10-07T09:00:00Z",
        Date.parse("2026-10-07T10:00:00Z"),
      ),
    ).toBeNull();
  });

  it("offers only relative, HTTP, or HTTPS context links", () => {
    expect(safeContextUrl("/docs/decision")).toBe(
      new URL("/docs/decision", window.location.origin).href,
    );
    expect(safeContextUrl("https://help.example.test/decision")).toBe(
      "https://help.example.test/decision",
    );
    expect(safeContextUrl("javascript:alert(1)")).toBeNull();
    expect(safeContextUrl("data:text/html,unsafe")).toBeNull();
  });
});
