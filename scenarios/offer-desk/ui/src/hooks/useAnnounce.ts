/**
 * @vrooliComponentSource react-component-library:useAnnounce
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 5e22d9a4-a909-4a40-8ca2-f3092fa16ffa
 * @vrooliComponentAppliedAt 2026-08-17T08:31:55Z
 * @vrooliComponentSourceSha256 093e394720edc0297f88be462e5c752d61007946d304681d3e56f9272980bbaa
 * @vrooliComponentDriftHash ddc51f0acad91cda918c76653afbfc9c38e88db2be65ed801bfe301c294bd8c6
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useCallback } from "react";
import {
  useLiveAnnouncer,
  type AnnouncementOptions,
} from "../../../../react-component-library/ui/src/services/LiveAnnouncer/versions/1.0.0/LiveAnnouncer";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) =>
      announcer.announce(message, options),
    [announcer],
  );
}
