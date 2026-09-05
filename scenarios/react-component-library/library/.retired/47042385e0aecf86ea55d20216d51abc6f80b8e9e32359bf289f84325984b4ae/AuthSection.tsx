/**
 * @libraryId react-component-library:AuthSection
 * @displayName AuthSection
 * @description Shared sign-in and sign-out actions.
 * @version 1.0.3
 * @tags ["monetization","auth"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:AuthSection */
import { AuthSection as BaseAuthSection } from "@vrooli/react-component-library/MonetizationAccount/1.0.0";
export type AuthSectionProps = {
  signedIn: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
  className?: string;
};
export const AuthSection = withClassName(function AuthSection(
  props: AuthSectionProps,
) {
  return <BaseAuthSection data-testid="monetization.auth-section" {...props} />;
});
