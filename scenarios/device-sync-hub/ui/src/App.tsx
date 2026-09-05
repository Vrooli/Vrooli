import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";
import { OnboardingScreen } from "./features/onboarding/OnboardingScreen";
import { useSession } from "./features/session/SessionProvider";

/**
 * Auth gate + app composition. Until this browser holds a device token it sees
 * the first-run OnboardingScreen (set up this hub as the owner, or join an
 * existing hub); once paired it sees the routed split-screen shell. Providers
 * (theme/session/realtime) wrap both so the session that onboarding writes is
 * the same one the shell reads.
 */
function Gate() {
  const { isPaired } = useSession();
  return isPaired ? <AppRouter /> : <OnboardingScreen />;
}

export default function App() {
  return (
    <Providers>
      <Gate />
    </Providers>
  );
}
