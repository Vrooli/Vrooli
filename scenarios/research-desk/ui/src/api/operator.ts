import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { ChangeSubject } from "@vrooli/proto-types/research-desk/v1/activity/activity_pb";
import { Delivery, DeliveryReason, EntryKind, NeedsYouKind, OperatorService, type Review } from "@vrooli/proto-types/research-desk/v1/operator/operator_pb";

import { serviceClient } from "./client";

/** Operator domain: the operator's words, answers, verdicts and the courier (SYSTEM.md §7, §9). */
const client = serviceClient(OperatorService);

export function useNeedsYou() {
  return useQuery({ queryKey: ["operator", "needs-you"], queryFn: () => client.listNeedsYou({}), refetchInterval: 30_000 });
}

export function useReviews() {
  return useQuery({ queryKey: ["operator", "reviews"], queryFn: () => client.listReviews({}) });
}

/**
 * The exact text a write would append, from the API's own dry run: the
 * confirmation shows this, never text rebuilt here. `key` names the
 * request; the query runs only while the confirmation is open.
 */
export function useDryRun<T extends { text: string }>(key: readonly unknown[], run: () => Promise<T>, enabled: boolean) {
  return useQuery({ queryKey: ["operator", "dry-run", ...key], queryFn: run, enabled, placeholderData: keepPreviousData, staleTime: 0, gcTime: 0 });
}

/** Each write takes `dryRun`: true returns the exact text and writes nothing. */
export const operatorApi = {
  answer: (commissionId: string, itemId: string, optionKey: string, note: string, dryRun = false) => client.answer({ commissionId, itemId, optionKey, note, dryRun }),
  addEntry: (commissionId: string, kind: EntryKind, text: string, dryRun = false) => client.addEntry({ commissionId, kind, text, dryRun }),
  enableSteering: (commissionId: string, dryRun = false) => client.enableSteering({ commissionId, dryRun }),
  addVerdict: (reviewId: string, topic: string, text: string, dryRun = false) => client.addVerdict({ reviewId, topic, text, dryRun }),
  setCourier: (target: { commissionId: string } | { global: true }, on: boolean) =>
    client.setCourier("global" in target ? { global: true, on } : { commissionId: target.commissionId, on }),
  /** Sends one test decision to the operator's phone and waits up to a minute for a device to receive it. */
  testCourier: () => client.testCourier({}),
  /** Records the operator's look in the desk's state folder (not the research home). */
  markSeen: (subject: ChangeSubject, id: string) => client.markSeen({ subject, id }),
  /** "Caught up": records a look at every commission and topic at once. */
  markAllSeen: () => client.markSeen({ all: true }),
};

export { Delivery, DeliveryReason, EntryKind, NeedsYouKind };
/** No topics, and the API says it could not tell which: unknown, not none. */
export function topicsUnknown(review: Review): boolean {
  return review.topics.length === 0 && review.topicsEvidence?.known === false;
}

export type { CourierStatus, Link as ItemLink, ListNeedsYouResponse, NeedsYouItem, Review, TestCourierResponse } from "@vrooli/proto-types/research-desk/v1/operator/operator_pb";
