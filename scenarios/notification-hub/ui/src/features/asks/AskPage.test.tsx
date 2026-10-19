import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { conversationsClient, identityClient } from "../../api/notifications";
import { AskPage } from "./AskPage";
import { orderedOptions, remainingUntil } from "./askView";

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
  return renderWithProviders(
    <Routes>
      <Route path="/asks/:askId" element={<AskPage />} />
    </Routes>,
    { initialEntries: ["/asks/ask-1"] },
  );
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
  afterEach(() => vi.clearAllMocks());

  it("shows ask details without login while keeping answer controls gated", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    renderAsk();

    expect(await screen.findByText("Close the BAS goal or re-aim at quality?")).toBeInTheDocument();
    expect(conversationsClient.getAsk).toHaveBeenCalledWith({ askId: "ask-1" });
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(screen.getByTestId(selectors.session.form)).toBeInTheDocument();
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
    expect(within(options[0]!).getByText(/Recommended/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.defaultNotice)).toHaveTextContent(
      "Re-aim at quality applies in 3 h",
    );
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
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    expect(conversationsClient.answer).toHaveBeenCalledWith({
      askId: "ask-1",
      answer: "close",
      note: "keep the macOS follow-up",
    });
  });

  it("sends exactly one answer, with the note, for one tap", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await screen.findAllByTestId(selectors.asks.option);
    await userEvent.type(
      screen.getByTestId(selectors.asks.note),
      "keep macOS last",
    );
    const [, close] = await screen.findAllByTestId(selectors.asks.option);
    await userEvent.click(close!);

    await waitFor(() =>
      expect(conversationsClient.answer).toHaveBeenCalledTimes(1),
    );
    expect(conversationsClient.answer).toHaveBeenCalledWith({
      askId: "ask-1",
      answer: "close",
      note: "keep macOS last",
    });
  });

  it("shows the ask without login while keeping answer controls gated", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({
      signedIn: false,
      subject: "",
      email: "",
    } as never);
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: pendingAsk(),
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.question)).toHaveTextContent(
      "Close the BAS goal or re-aim at quality?",
    );
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
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
});
