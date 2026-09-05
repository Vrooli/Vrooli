interface StreamingDegradationNoticeProps {
  notice: string | null;
}

/** A turn-scoped status, not a service-down indicator. */
export function StreamingDegradationNotice({ notice }: StreamingDegradationNoticeProps) {
  if (!notice) return null;
  return <div role="status" data-testid="streaming-degradation-notice">{notice}</div>;
}
