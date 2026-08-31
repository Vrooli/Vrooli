/**
 * @libraryId react-component-library:useAnnounce
 * @displayName useAnnounce
 * @description A live-region primitive that queues, deduplicates, and times screen-reader messages through the shared announcer rather than creating competing regions.
 * @version 1.0.2
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-announce */
import { useCallback } from "react";
import {
  useLiveAnnouncer,
  type AnnouncementOptions,
} from "@vrooli/react-component-library/LiveAnnouncer/1";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) => announcer.announce(message, options),
    [announcer],
  );
}
