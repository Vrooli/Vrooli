import { create } from "@bufbuild/protobuf";
import { ListAsksResponseSchema, type Ask, type AskFacetCount, type ListAsksRequest, type ListAsksResponse } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

/** A page of decisions as `ListAsks` returns it. */
export interface AsksPage {
  asks: Ask[];
  nextCursor?: string;
  totalAwaiting?: number;
  /** The filter choices the server counted, as `[value, count]` pairs. Absent means the server sent none. */
  facets?: { urgencies?: [string, number][]; senders?: [string, number][] };
}

const facetCounts = (pairs: [string, number][] = []): Pick<AskFacetCount, "value" | "count">[] => pairs.map(([value, count]) => ({ value, count }));

/** The shell's own read of what is waiting: one awaiting row, no cursor, whose `totalAwaiting` is the count. */
const isCountRead = (request: Partial<ListAsksRequest>) =>
  request.limit === 1 && !("cursor" in request) && request.stateFilter === "awaiting";

/**
 * An implementation for a mocked `ListAsks` that answers the shell's waiting
 * count (`awaiting` decisions, none by default) and everything else with
 * `handler`. Without it, a test that scripts list pages one after another would
 * have the count read consume them.
 */
export function listAsksFor(handler: (request: Partial<ListAsksRequest>) => AsksPage | Promise<AsksPage>, awaiting = 0) {
  const respond = (page: AsksPage): ListAsksResponse =>
    create(ListAsksResponseSchema, {
      asks: page.asks, nextCursor: page.nextCursor ?? "", totalAwaiting: page.totalAwaiting ?? 0,
      ...(page.facets ? { facets: { urgencies: facetCounts(page.facets.urgencies), senders: facetCounts(page.facets.senders) } } : {}),
    });
  return async (request: Partial<ListAsksRequest>): Promise<ListAsksResponse> =>
    isCountRead(request) ? respond({ asks: [], totalAwaiting: awaiting }) : respond(await handler(request));
}

/** Answer successive list reads with each result in turn; an Error rejects that read. */
export function asksInOrder(...results: (AsksPage | Error)[]) {
  const queue = [...results];
  return listAsksFor(() => {
    const next = queue.shift();
    if (next instanceof Error) return Promise.reject(next);
    return next ?? { asks: [], nextCursor: "" };
  });
}
