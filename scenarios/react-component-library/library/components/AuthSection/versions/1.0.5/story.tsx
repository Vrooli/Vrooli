import { AuthSection } from "./AuthSection";

export function AuthStory() {
  return <AuthSection signedIn={false} onSignIn={() => undefined} onSignOut={() => undefined} />;
}
