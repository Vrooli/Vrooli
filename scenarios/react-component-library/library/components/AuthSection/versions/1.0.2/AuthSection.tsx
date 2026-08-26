/**
 * @libraryId react-component-library:AuthSection
 * @displayName AuthSection
 * @description Shared sign-in and sign-out actions.
 * @version 1.0.2
 * @tags ["monetization","auth"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AuthSection */
import { AuthSection as BaseAuthSection } from "../../../MonetizationAccount/versions/1.0.0/MonetizationAccount";
export type AuthSectionProps = {
  signedIn: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
  className?: string;
};
export function AuthSection(props: AuthSectionProps) {
  return <BaseAuthSection {...props} />;
}
