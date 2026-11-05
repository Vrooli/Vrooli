/** @vrooliComponentSource hooks.use-announce */
import { useCallback } from "react";
import {
  useLiveAnnouncer,
  type AnnouncementOptions,
} from "../../../../services/LiveAnnouncer/versions/1.0.0/LiveAnnouncer";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) =>
      announcer.announce(message, options),
    [announcer],
  );
}
