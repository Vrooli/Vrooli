import { useAnnounce } from "./useAnnounce";
import { LiveAnnouncer } from "@vrooli/react-component-library/LiveAnnouncer/1";

export function Default({ log }: StoryHarnessProps) {
  const announce = useAnnounce();
  return (
    <LiveAnnouncer data-testid="hooks.use-announce">
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
