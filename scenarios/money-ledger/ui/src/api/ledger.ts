import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { BooksService, JournalService, ListAccountsRequestSchema, ListBooksRequestSchema, ListPostingsRequestSchema, PositionRequestSchema, PositionService, StatementRequestSchema } from "@vrooli/proto-types/money-ledger/v1/ledger/ledger_pb";
import { IngestService, ListAdaptersRequestSchema } from "@vrooli/proto-types/money-ledger/v1/ingest/ingest_pb";
import { API_BASE, transport } from "./client";

export const positionClient = createClient(PositionService, transport);
export const booksClient = createClient(BooksService, transport);
export const journalClient = createClient(JournalService, transport);
export const ingestClient = createClient(IngestService, transport);
export const configuredBookId = () => new URLSearchParams(window.location.search).get("book_id") ?? window.localStorage.getItem("money-ledger.book-id") ?? "";
export async function fetchPosition() {
  const bookId = configuredBookId();
  if (!bookId) return null;
  return positionClient.getPosition(create(PositionRequestSchema, { bookId }));
}
export function fetchBooks() { return booksClient.listBooks(create(ListBooksRequestSchema)); }
export function fetchAccounts(bookId: string) { return booksClient.listAccounts(create(ListAccountsRequestSchema, { bookId })); }
export function fetchPostings(bookId: string) { return journalClient.listPostings(create(ListPostingsRequestSchema, { bookId, limit: 100 })); }
export function fetchAdapters() { return ingestClient.listAdapters(create(ListAdaptersRequestSchema)); }
export function fetchStatement(bookId: string, from = "", to = "") { return positionClient.getStatement(create(StatementRequestSchema, { bookId, from, to })); }
export { API_BASE };
