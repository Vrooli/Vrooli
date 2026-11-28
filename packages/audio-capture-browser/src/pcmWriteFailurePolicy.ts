export interface PcmWriteFailureClassification {
  readonly reason: "sequence_mismatch" | "quota_exhausted" | "storage_transaction" | "storage_write";
  readonly sequenceDetail: readonly [expectedSequence: string, expectedSample: string, actualSequence: string, actualSample: string] | null;
}

/** Classify journal write failures into stable, metadata-only diagnostic tags. */
export function classifyPcmWriteFailure(causeName: string, message: string): PcmWriteFailureClassification {
  const sequenceMatch = message.match(/expected sequence (\d+) at sample (\d+), got sequence (\d+) at sample (\d+)/i);
  const reason = /contiguous sequence identities/i.test(message)
    ? "sequence_mismatch"
    : /quota exhausted|quota exceeded/i.test(message)
      ? "quota_exhausted"
      : /indexeddb|transaction|abort/i.test(`${causeName} ${message}`)
        ? "storage_transaction"
        : "storage_write";
  const sequenceDetail = sequenceMatch
    ? [sequenceMatch[1]!, sequenceMatch[2]!, sequenceMatch[3]!, sequenceMatch[4]!] as const
    : null;
  return {
    reason,
    sequenceDetail,
  };
}
