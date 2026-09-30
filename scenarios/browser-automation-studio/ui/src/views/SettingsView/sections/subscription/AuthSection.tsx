import * as MonetizationAccount from '@components/MonetizationAccount';
import { useAuthStore, useIsAuthenticated } from '@stores/authStore';

/** BAS adapter for the RCL account primitive; auth state remains scenario-owned. */
export function AuthSection() {
  const { signIn, signOut, error } = useAuthStore();
  const signedIn = useIsAuthenticated();

  return (
    <div className="space-y-2">
      <MonetizationAccount.AuthSection
        signedIn={signedIn}
        onSignIn={() => void signIn()}
        onSignOut={() => void signOut()}
      />
      {error && <p className="text-sm text-red-400">{error}</p>}
    </div>
  );
}

export default AuthSection;
