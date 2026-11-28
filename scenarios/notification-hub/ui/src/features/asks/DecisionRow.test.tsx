import { beforeEach, describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { makeAsk } from "../packet/mocks/askFactory";
import { DecisionRow } from "./DecisionRow";

const SENDER = "Steward";

const renderRow = (ask = makeAsk({ id: "ask-1", question: "Choose the next base" })) => {
  const view = renderWithProviders(
    <ul>
      <DecisionRow ask={ask} queue search="" from="/asks" onOpen={() => undefined} />
    </ul>,
  );
  return view;
};

describe("DecisionRow", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("shows the specific title, its source and age, and the readiness chip", () => {
    const ask = makeAsk({
      id: "ask-1",
      title: "Stop the deletion floors asking twice",
      question: "Which deletion floor should change?",
      sender: "Steward",
      state: "pending",
      createdAt: "2026-01-01T00:00:00.000Z",
      qualityGrade: "needs_work",
    });
    renderRow(ask);
    expect(screen.getByText(ask.title)).toBeInTheDocument();
    expect(screen.getByText(SENDER)).toBeInTheDocument();
    expect(screen.getByText(en.asks.row.readiness.needsWork)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.item)).toHaveTextContent(SENDER);
  });

  it.each([
    ["needs_work", en.asks.row.readiness.needsWork],
    ["refused", en.asks.row.readiness.returned],
  ] as const)("names the %s grade without relying on colour", (qualityGrade, label) => {
    renderRow(makeAsk({ qualityGrade }));
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("keeps a Ready row quiet: the grade is not a fourth chip", () => {
    renderRow(makeAsk({ qualityGrade: "ready" }));
    expect(screen.queryByText(en.asks.row.readiness.ready)).not.toBeInTheDocument();
  });

  it("draws no readiness chip when the Hub never assessed the packet", () => {
    renderRow(makeAsk({ qualityGrade: "" }));
    expect(screen.queryByText(en.asks.row.readiness.ready)).not.toBeInTheDocument();
    expect(screen.queryByText(en.asks.row.readiness.needsWork)).not.toBeInTheDocument();
    expect(screen.queryByText(en.asks.row.readiness.returned)).not.toBeInTheDocument();
  });

  it("keeps the close time in the compact meta line", () => {
    renderRow(makeAsk({
      closingSoon: true,
      deadline: new Date(Date.now() + 3 * 3600_000).toISOString(),
    }));
    expect(screen.getByText(/^Closes /)).toBeInTheDocument();
  });
});
