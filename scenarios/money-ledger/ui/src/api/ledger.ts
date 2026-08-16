import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  BooksService,
  CreateAccountRequestSchema,
  CreateBookRequestSchema,
  DeclareGoalRequestSchema,
  GoalSchema,
  JournalService,
  ListAccountsRequestSchema,
  ListBooksRequestSchema,
  ListGoalsRequestSchema,
  ListPostingsRequestSchema,
  PositionRequestSchema,
  PositionService,
  ReversePostingRequestSchema,
  StatementRequestSchema,
  SustainPeriodUnit,
  TransferRequestSchema,
} from "@vrooli/proto-types/money-ledger/v1/ledger/ledger_pb";
import { AdapterKind, AdapterSchema, ImportFileRequestSchema, IngestEventRequestSchema, IngestService, ListAdaptersRequestSchema, RegisterAdapterRequestSchema, RunAdapterRequestSchema } from "@vrooli/proto-types/money-ledger/v1/ingest/ingest_pb";
import { Basis, MoneyEventSchema } from "@vrooli/proto-types/money-ledger/v1/shared/ledger_types_pb";
import { API_BASE, transport } from "./client";

export const positionClient = createClient(PositionService, transport);
export const booksClient = createClient(BooksService, transport);
export const journalClient = createClient(JournalService, transport);
export const ingestClient = createClient(IngestService, transport);
export const configuredBookId = () => new URLSearchParams(window.location.search).get("book_id") ?? window.localStorage.getItem("money-ledger.book-id") ?? "";
export async function fetchPosition(bookIdOverride = configuredBookId(), from = "", to = "") {
  const bookId = bookIdOverride;
  if (!bookId) return null;
  return positionClient.getPosition(create(PositionRequestSchema, { bookId, from, to }));
}
export function fetchBooks() { return booksClient.listBooks(create(ListBooksRequestSchema)); }
export function fetchAccounts(bookId: string) { return booksClient.listAccounts(create(ListAccountsRequestSchema, { bookId })); }
export function fetchPostings(bookId: string) { return journalClient.listPostings(create(ListPostingsRequestSchema, { bookId, limit: 100 })); }
export function fetchAdapters() { return ingestClient.listAdapters(create(ListAdaptersRequestSchema)); }
export function fetchGoals(bookId: string) { return positionClient.listGoals(create(ListGoalsRequestSchema, { bookId })); }
export function fetchStatement(bookId: string, from = "", to = "") { return positionClient.getStatement(create(StatementRequestSchema, { bookId, from, to })); }

export function registerAdapter(input: { id: string; name: string; kind: AdapterKind; enabled?: boolean }) {
  return ingestClient.registerAdapter(create(RegisterAdapterRequestSchema, {
    adapter: create(AdapterSchema, { id: input.id, name: input.name, kind: input.kind, enabled: input.enabled ?? true }),
  }));
}

export function registerManualAdapter() { return registerAdapter({ id: "manual", name: "Manual entry", kind: AdapterKind.MANUAL }); }

export function runAdapter(adapterId: string) {
  return ingestClient.runAdapter(create(RunAdapterRequestSchema, { adapterId }));
}

export function importFile(adapterId: string, csv: Uint8Array) {
  return ingestClient.importFile(create(ImportFileRequestSchema, { adapterId, csv }));
}

export function declareGoal(input: { bookId: string; name: string; metric: string; comparator: string; thresholdMinor: bigint; sustainPeriods: number; periodUnit: SustainPeriodUnit; bufferMultiple?: number; comparandMetric?: string }) {
  return positionClient.declareGoal(create(DeclareGoalRequestSchema, {
    goal: create(GoalSchema, {
      bookId: input.bookId,
      name: input.name,
      metric: input.metric,
      comparator: input.comparator,
      thresholdMinor: input.thresholdMinor,
      sustainPeriods: input.sustainPeriods,
      sustainPeriodUnit: input.periodUnit,
      bufferMultiple: input.bufferMultiple ?? 1,
      comparandMetric: input.comparandMetric ?? "",
    }),
  }));
}

export function createBook(name: string, currency: string) {
  return booksClient.createBook(create(CreateBookRequestSchema, { name, currency }));
}

export function createAccount(bookId: string, name: string, kind: string) {
  return booksClient.createAccount(create(CreateAccountRequestSchema, { bookId, name, kind }));
}

export function ingestEvent(input: {
  externalId: string;
  accountId: string;
  bookId: string;
  amountMinor: bigint;
  currency: string;
  occurredAt: Date;
  description: string;
}) {
  const event = create(MoneyEventSchema, {
    externalId: input.externalId,
    adapterId: "manual",
    accountId: input.accountId,
    bookId: input.bookId,
    amountMinor: input.amountMinor,
    currency: input.currency,
    occurredAt: timestampFromDate(input.occurredAt),
    fetchedAt: timestampFromDate(new Date()),
    basis: Basis.OPERATOR_ASSERTED,
    description: input.description,
  });
  return ingestClient.ingestEvent(create(IngestEventRequestSchema, { event }));
}

export function reversePosting(postingId: string, reason: string) {
  return journalClient.reversePosting(create(ReversePostingRequestSchema, { postingId, reason }));
}

export function transfer(input: {
  fromAccountId: string;
  toAccountId: string;
  amountMinor: bigint;
  currency: string;
  externalId: string;
  description: string;
  occurredAt: Date;
}) {
  return journalClient.transfer(create(TransferRequestSchema, {
    ...input,
    occurredAt: timestampFromDate(input.occurredAt),
  }));
}
export { API_BASE };
