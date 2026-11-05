import { useSession } from "../features/session/SessionGate";
import { useLocation } from "react-router-dom";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { ReachSettings } from "../features/reach/ReachSettings";
import { DeviceAppsPanel } from "../features/reach/DeviceAppsPanel";

/** Focused Settings destination for device and channel reachability. */
export function ReachPage() {
  const session = useSession();
  const location = useLocation();
  const signedIn = Boolean(session.data?.signedIn);
  return (
    <>
      <ReachSettings
        signedIn={signedIn}
        focusedEntry
        preferencesHref={preserveRoutedTestMode("/settings?view=preferences", location.search)}
      />
      {signedIn && <DeviceAppsPanel />}
    </>
  );
}
