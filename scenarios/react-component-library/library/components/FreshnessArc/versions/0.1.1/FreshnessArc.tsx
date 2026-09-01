/**
 * @libraryId react-component-library:FreshnessArc
 * @displayName FreshnessArc
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export default function FreshnessArc({
  ageSeconds = 0,
  ttlSeconds = 60,
}: {
  ageSeconds?: number;
  ttlSeconds?: number;
}) {
  const ratio = Math.max(0, Math.min(1, 1 - ageSeconds / Math.max(1, ttlSeconds)));
  return (
    <div
      role="progressbar"
      aria-valuenow={Math.round(ratio * 100)}
      aria-label="Freshness"
      style={{ "--freshness": ratio } as React.CSSProperties}
    />
  );
}
