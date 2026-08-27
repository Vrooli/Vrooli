import { AuthSection as AccountAuthSection } from "./MonetizationAccount";

export function AuthSection() {
  return (
    <AccountAuthSection
      signedIn={false}
      onSignIn={() => undefined}
      onSignOut={() => undefined}
    />
  );
}
