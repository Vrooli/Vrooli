/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 3.0.0
 * @vrooliComponentAdoption 1facf313-4a56-4ab5-8731-0d468b0a929b
 * @vrooliComponentAppliedAt 2026-08-05T04:35:59Z
 * @vrooliComponentSourceSha256 674e3a23ffed846b12fd06c4a23104f12d57560c87d699e0d8e3f8c0a21a4c3a
 * @vrooliComponentDriftHash 674e3a23ffed846b12fd06c4a23104f12d57560c87d699e0d8e3f8c0a21a4c3a
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { SVGProps } from "react";

export type VoiceInputGlyphKind = "alert" | "loader" | "mic" | "close";

const paths: Record<VoiceInputGlyphKind, JSX.Element> = {
  mic: <><rect x="8" y="3" width="8" height="12" rx="4" /><path d="M5 11a7 7 0 0 0 14 0M12 21v-4M9 21h6" /></>,
  loader: <><path d="M12 2a10 10 0 1 0 10 10" /><path d="M12 2v4" /></>,
  alert: <><path d="M10.3 3.7 2.2 18a2 2 0 0 0 1.7 3h16.2a2 2 0 0 0 1.7-3L13.7 3.7a2 2 0 0 0-3.4 0Z" /><path d="M12 9v4M12 17h.01" /></>,
  close: <><path d="m6 6 12 12M18 6 6 18" /></>,
};

export function VoiceInputButtonGlyph({ kind, className = "", ...props }: SVGProps<SVGSVGElement> & { kind: VoiceInputGlyphKind }) {
  return <svg {...props} aria-hidden="true" className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{paths[kind]}</svg>;
}
