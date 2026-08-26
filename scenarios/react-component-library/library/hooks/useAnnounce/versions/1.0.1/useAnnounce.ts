/**
 * @libraryId react-component-library:useAnnounce
 * @displayName useAnnounce
 * @description Production-ready useAnnounce hook with SSR-safe lifecycle behavior.
 * @version 1.0.1
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-announce */
import { useCallback } from "react";
import {
  useLiveAnnouncer,
  type AnnouncementOptions,
} from "@vrooli/react-component-library/LiveAnnouncer/1.0.0";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) => announcer.announce(message, options),
    [announcer],
  );
}
