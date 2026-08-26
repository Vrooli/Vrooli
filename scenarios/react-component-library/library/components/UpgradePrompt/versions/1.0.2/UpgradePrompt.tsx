/**
 * @libraryId react-component-library:UpgradePrompt
 * @displayName UpgradePrompt
 * @description Hosted upgrade prompt for a gated capability.
 * @version 1.0.2
 * @tags ["monetization","upgrade"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:UpgradePrompt */
import { UpgradePrompt as BaseUpgradePrompt } from "../../../MonetizationAccount/versions/1.0.0/MonetizationAccount";
export type UpgradePromptProps = {
  feature: string;
  requiredPlan: string;
  href?: string;
  className?: string;
};
export function UpgradePrompt(props: UpgradePromptProps) {
  return <BaseUpgradePrompt {...props} />;
}
