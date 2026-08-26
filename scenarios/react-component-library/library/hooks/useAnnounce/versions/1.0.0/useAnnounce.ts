/** @vrooliComponentSource hooks.use-announce */
import { useCallback } from "react";
import {
  useLiveAnnouncer,
  type AnnouncementOptions,
} from "@vrooli/react-component-library/LiveAnnouncer/1.0.0";

export function useAnnounce() {
  const announcer = useLiveAnnouncer();
  return useCallback(
    (message: string, options?: AnnouncementOptions) =>
      announcer.announce(message, options),
    [announcer],
  );
}
