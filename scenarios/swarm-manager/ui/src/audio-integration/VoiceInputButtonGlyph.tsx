/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 4.0.0
 * @vrooliComponentAdoption b11fce87-57ac-4860-b70f-75c25987167f
 * @vrooliComponentAppliedAt 2026-08-09T14:56:12Z
 * @vrooliComponentSourceSha256 917eef020c6f5479c4d7bbaa930ccf0276bfb4270d3fe5b0aceb1e0ad7cb5c5d
 * @vrooliComponentDriftHash 917eef020c6f5479c4d7bbaa930ccf0276bfb4270d3fe5b0aceb1e0ad7cb5c5d
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode, SVGProps } from "react";

export type VoiceInputGlyphKind = "alert" | "loader" | "mic";

const paths: Record<VoiceInputGlyphKind, ReactNode> = {
  mic: (
    <>
      <rect x="8" y="3" width="8" height="12" rx="4" />
      <path d="M5 11a7 7 0 0 0 14 0M12 21v-4M9 21h6" />
    </>
  ),
  loader: (
    <>
      <path d="M12 2a10 10 0 1 0 10 10" />
      <path d="M12 2v4" />
    </>
  ),
  alert: (
    <>
      <path d="M10.3 3.7 2.2 18a2 2 0 0 0 1.7 3h16.2a2 2 0 0 0 1.7-3L13.7 3.7a2 2 0 0 0-3.4 0Z" />
      <path d="M12 9v4M12 17h.01" />
    </>
  ),
};

export function VoiceInputButtonGlyph({
  kind,
  className = "",
  ...props
}: SVGProps<SVGSVGElement> & { kind: VoiceInputGlyphKind }) {
  return (
    <svg
      {...props}
      aria-hidden="true"
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {paths[kind]}
    </svg>
  );
}
