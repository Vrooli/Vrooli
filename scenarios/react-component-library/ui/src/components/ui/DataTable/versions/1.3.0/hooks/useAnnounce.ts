/**
 * @vrooliComponentSource react-component-library:useAnnounce
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption ea21d62b-0215-49e0-8816-95046b0c66f5
 * @vrooliComponentAppliedAt 2026-08-18T01:12:39Z
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
} from "../../../../services/LiveAnnouncer/versions/1.0.0/LiveAnnouncer";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) => announcer.announce(message, options),
    [announcer],
  );
}
