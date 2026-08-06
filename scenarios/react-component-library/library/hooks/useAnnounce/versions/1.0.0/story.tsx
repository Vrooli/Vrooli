import { useAnnounce } from "./useAnnounce";
import { LiveAnnouncer } from "../../../../services/LiveAnnouncer/versions/1.0.0/LiveAnnouncer";

export function Default({ log }: StoryHarnessProps) {
  const announce = useAnnounce();
  return (
    <LiveAnnouncer>
      <button
        type="button"
        onClick={() => {
          announce("Updated");
          log("announced");
        }}
      >
        Announce
      </button>
    </LiveAnnouncer>
  );
}
