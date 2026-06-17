import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";
import { JoinScreen } from "./features/session/JoinScreen";
import { useSession } from "./features/session/SessionProvider";

/**
 * Auth gate + app composition. Until this browser holds a device token it sees
 * the JoinScreen (redeem a pairing code / request approval); once paired it
 * sees the routed split-screen shell. Providers (theme/session/realtime) wrap
 * both so the session that JoinScreen writes is the same one the shell reads.
 */
function Gate() {
  const { isPaired } = useSession();
  return isPaired ? <AppRouter /> : <JoinScreen />;
}

export default function App() {
  return (
    <Providers>
      <Gate />
    </Providers>
  );
}
