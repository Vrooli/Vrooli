import { useSession } from "../features/session/SessionGate";
import { useLocation } from "react-router-dom";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { ReachSettings } from "../features/reach/ReachSettings";

/** Focused Settings destination for device and channel reachability. */
export function ReachPage() {
  const session = useSession();
  const location = useLocation();
  return (
    <ReachSettings
      signedIn={Boolean(session.data?.signedIn)}
      focusedEntry
      preferencesHref={preserveRoutedTestMode("/settings?view=preferences", location.search)}
    />
  );
}
