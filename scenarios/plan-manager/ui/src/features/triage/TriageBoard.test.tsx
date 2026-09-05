/**
 * TriageBoard tests — candidate list, empty state, promote/dismiss, and
 * axe-clean structure. api/log is mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  FindingTriage,
  LogEntrySchema,
  LogEntryType,
  type LogEntry,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const listEntries = vi.fn();
const promoteEntry = vi.fn();
const updateEntry = vi.fn();

vi.mock("../../api/log", () => ({
  listEntries: (...a: unknown[]) => listEntries(...a),
  promoteEntry: (...a: unknown[]) => promoteEntry(...a),
  updateEntry: (...a: unknown[]) => updateEntry(...a),
}));

import { TriageBoard } from "./TriageBoard";

const finding = create(LogEntrySchema, {
  id: "f1",
  type: LogEntryType.FINDING,
  title: "Possible nil deref",
  detail: "in handler",
  phaseId: "p1",
  triage: FindingTriage.CANDIDATE,
  createdAt: "2026-06-25T10:00:00Z",
});

const listResult = (entries: LogEntry[]) => ({ entries, summary: undefined, step: undefined });

describe("TriageBoard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state with no candidates", async () => {
    listEntries.mockResolvedValue(listResult([]));
    renderWithProviders(<TriageBoard />);
    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.triage.list}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("lists candidate findings", async () => {
    listEntries.mockResolvedValue(listResult([finding]));
    renderWithProviders(<TriageBoard />);
    expect(await screen.findByTestId(selectors.triage.row({ id: "f1" }))).toBeInTheDocument();
    await waitFor(() => {
      expect(listEntries).toHaveBeenCalledWith({
        type: LogEntryType.FINDING,
        triage: FindingTriage.CANDIDATE,
      });
    });
  });

  it("renders findings without optional detail, phase, or timestamp", async () => {
    listEntries.mockResolvedValue(
      listResult([create(LogEntrySchema, { id: "f-min", type: LogEntryType.FINDING, title: "Minimal" })]),
    );
    renderWithProviders(<TriageBoard />);
    const row = await screen.findByTestId(selectors.triage.row({ id: "f-min" }));
    expect(row).toHaveTextContent("Minimal");
    expect(row).not.toHaveTextContent("Phase");
    expect(row).not.toHaveTextContent("Recorded");
  });

  it("promotes a finding to a bug", async () => {
    const user = userEvent.setup();
    listEntries.mockResolvedValue(listResult([finding]));
    promoteEntry.mockResolvedValue({ entry: create(LogEntrySchema, { id: "b1" }), source: finding, step: undefined });

    renderWithProviders(<TriageBoard />);
    await user.click(await screen.findByTestId(selectors.triage.promote({ id: "f1" })));
    await waitFor(() => {
      expect(promoteEntry).toHaveBeenCalledWith({ id: "f1", toType: LogEntryType.BUG_REPORT });
    });
  });

  it("dismisses a finding", async () => {
    const user = userEvent.setup();
    listEntries.mockResolvedValue(listResult([finding]));
    updateEntry.mockResolvedValue({
      entry: create(LogEntrySchema, { id: "f1", triage: FindingTriage.DISMISSED }),
      step: undefined,
    });

    renderWithProviders(<TriageBoard />);
    await user.click(await screen.findByTestId(selectors.triage.dismiss({ id: "f1" })));
    await waitFor(() => {
      expect(updateEntry).toHaveBeenCalledWith({ id: "f1", triage: FindingTriage.DISMISSED });
    });
  });

  it("renders the list without axe violations", async () => {
    listEntries.mockResolvedValue(listResult([finding]));
    const { container } = renderWithProviders(<TriageBoard />);
    await screen.findByTestId(selectors.triage.row({ id: "f1" }));
    await expectNoA11yViolations(container);
  });
});
