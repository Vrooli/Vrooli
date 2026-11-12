import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { ChangeSubject } from "@vrooli/proto-types/research-desk/v1/activity/activity_pb";
import { Delivery, DeliveryReason, EntryKind, OperatorService } from "@vrooli/proto-types/research-desk/v1/operator/operator_pb";

import { serviceClient } from "./client";

/** Operator domain: the operator's words, answers, verdicts and the courier (SYSTEM.md §7, §10). */
const client = serviceClient(OperatorService);

export function useNeedsYou() {
  return useQuery({ queryKey: ["operator", "needs-you"], queryFn: () => client.listNeedsYou({}), refetchInterval: 30_000 });
}

export function useReviews() {
  return useQuery({ queryKey: ["operator", "reviews"], queryFn: () => client.listReviews({}) });
}

/**
 * The exact text a write would append, from the API's own dry run (rulings
 * P5): the confirmation shows this, never text rebuilt here. `key` names the
 * request; the query runs only while the confirmation is open.
 */
export function useDryRun(key: readonly unknown[], run: () => Promise<{ text: string }>, enabled: boolean) {
  return useQuery({ queryKey: ["operator", "dry-run", ...key], queryFn: run, enabled, placeholderData: keepPreviousData, staleTime: 0, gcTime: 0 });
}

/** Each write takes `dryRun`: true returns the exact text and writes nothing (rulings P5). */
export const operatorApi = {
  answer: (commissionId: string, itemId: string, optionKey: string, note: string, dryRun = false) => client.answer({ commissionId, itemId, optionKey, note, dryRun }),
  addEntry: (commissionId: string, kind: EntryKind, text: string, dryRun = false) => client.addEntry({ commissionId, kind, text, dryRun }),
  enableSteering: (commissionId: string) => client.enableSteering({ commissionId }),
  addVerdict: (reviewId: string, topic: string, text: string, dryRun = false) => client.addVerdict({ reviewId, topic, text, dryRun }),
  setCourier: (target: { commissionId: string } | { global: true }, on: boolean) =>
    client.setCourier("global" in target ? { global: true, on } : { commissionId: target.commissionId, on }),
  /** Sends one test decision to the operator's phone and waits up to a minute for a device to receive it (rulings F15). */
  testCourier: () => client.testCourier({}),
  /** Records the operator's look in the desk's state folder (not the research home). */
  markSeen: (subject: ChangeSubject, id: string) => client.markSeen({ subject, id }),
  /** "Caught up": records a look at every commission and topic at once (rulings P7). */
  markAllSeen: () => client.markSeen({ all: true }),
};

export { Delivery, DeliveryReason, EntryKind };
export type { CourierStatus, Link as ItemLink, NeedsYouItem, Review, TestCourierResponse } from "@vrooli/proto-types/research-desk/v1/operator/operator_pb";
