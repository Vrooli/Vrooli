/**
 * @libraryId react-component-library:AuthSection
 * @displayName AuthSection
 * @version 1.0.5
 * @tags ["monetization","auth"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:AuthSection */
import { AuthSection as BaseAuthSection } from "@vrooli/react-component-library/MonetizationAccount/2";
export type AuthSectionProps = {
  signedIn: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
  className?: string;
};
export const AuthSection = withClassName(function AuthSection(props: AuthSectionProps) {
  return <BaseAuthSection data-testid="monetization.auth-section" {...props} />;
});
