import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { conversationsClient } from "../../api/notifications";
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
  });
  afterEach(() => vi.clearAllMocks());

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
