import { beforeEach, describe, expect, it, vi } from "vitest";
import { AdapterKind } from "@vrooli/proto-types/money-ledger/v1/ingest/ingest_pb";
import { SustainPeriodUnit } from "@vrooli/proto-types/money-ledger/v1/ledger/ledger_pb";

const client = vi.hoisted(() => ({
  getPosition: vi.fn(),
  listBooks: vi.fn(),
  listAccounts: vi.fn(),
  listPostings: vi.fn(),
  listAdapters: vi.fn(),
  listGoals: vi.fn(),
  getStatement: vi.fn(),
  registerAdapter: vi.fn(),
  runAdapter: vi.fn(),
  importFile: vi.fn(),
  declareGoal: vi.fn(),
  createBook: vi.fn(),
  createAccount: vi.fn(),
  ingestEvent: vi.fn(),
  reversePosting: vi.fn(),
  transfer: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => client),
}));

vi.mock("./client", () => ({
  API_BASE: "http://money-ledger.test",
  transport: {},
}));

import {
  configuredBookId,
  createAccount,
  createBook,
  declareGoal,
  fetchAccounts,
  fetchAdapters,
  fetchBooks,
  fetchGoals,
  fetchPosition,
  fetchPostings,
  fetchStatement,
  importFile,
  ingestEvent,
  registerAdapter,
  registerManualAdapter,
  reversePosting,
  runAdapter,
  transfer,
} from "./ledger";

describe("ledger Connect wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.replaceState({}, "", "/");
    window.localStorage.clear();
    for (const method of Object.values(client)) method.mockResolvedValue({ ok: true });
  });

  it("builds typed requests for reads and configured-book resolution", async () => {
    expect(configuredBookId()).toBe("");
    window.history.replaceState({}, "", "/?book_id=query-book");
    expect(configuredBookId()).toBe("query-book");
    window.history.replaceState({}, "", "/");
    window.localStorage.setItem("money-ledger.book-id", "stored-book");
    expect(configuredBookId()).toBe("stored-book");
    expect(await fetchPosition()).toEqual({ ok: true });
    expect(await fetchPosition("explicit-book", "2026-01-01", "2026-01-31")).toEqual({ ok: true });
    expect(client.getPosition).toHaveBeenCalledTimes(2);
    expect(await fetchBooks()).toEqual({ ok: true });
    expect(await fetchAccounts("book-1")).toEqual({ ok: true });
    expect(await fetchPostings("book-1")).toEqual({ ok: true });
    expect(await fetchAdapters()).toEqual({ ok: true });
    expect(await fetchGoals("book-1")).toEqual({ ok: true });
    expect(await fetchStatement("book-1", "2026-01-01", "2026-01-31")).toEqual({ ok: true });
  });

  it("builds typed requests for writes and ingestion operations", async () => {
    const occurredAt = new Date("2026-08-16T12:00:00Z");
    expect(await registerAdapter({ id: "csv", name: "CSV", kind: AdapterKind.FILE })).toEqual({ ok: true });
    expect(await registerManualAdapter()).toEqual({ ok: true });
    expect(await runAdapter("csv")).toEqual({ ok: true });
    expect(await importFile("csv", new Uint8Array([1, 2, 3]))).toEqual({ ok: true });
    expect(await declareGoal({
      bookId: "book-1",
      name: "Runway",
      metric: "cash",
      comparator: ">=",
      thresholdMinor: 1000n,
      sustainPeriods: 2,
      periodUnit: SustainPeriodUnit.MONTH,
      bufferMultiple: 1.5,
      comparandMetric: "expense",
    })).toEqual({ ok: true });
    expect(await createBook("Operating", "USD")).toEqual({ ok: true });
    expect(await createAccount("book-1", "Cash", "ASSET")).toEqual({ ok: true });
    expect(await ingestEvent({
      externalId: "event-1",
      accountId: "cash",
      bookId: "book-1",
      amountMinor: 2500n,
      currency: "USD",
      occurredAt,
      description: "Sale",
    })).toEqual({ ok: true });
    expect(await reversePosting("posting-1", "Wrong amount")).toEqual({ ok: true });
    expect(await transfer({
      fromAccountId: "cash",
      toAccountId: "reserve",
      amountMinor: 500n,
      currency: "USD",
      externalId: "transfer-1",
      description: "Reserve",
      occurredAt,
    })).toEqual({ ok: true });
    expect(client.registerAdapter).toHaveBeenCalledTimes(2);
    expect(client.importFile).toHaveBeenCalledWith(expect.objectContaining({ adapterId: "csv", csv: new Uint8Array([1, 2, 3]) }));
    expect(client.ingestEvent).toHaveBeenCalledWith(expect.objectContaining({ event: expect.objectContaining({ adapterId: "manual", externalId: "event-1" }) }));
  });
});
