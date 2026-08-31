/**
 * @libraryId react-component-library:UpgradePrompt
 * @displayName UpgradePrompt
 * @version 1.0.5
 * @tags ["monetization","upgrade"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:UpgradePrompt */
import { UpgradePrompt as BaseUpgradePrompt } from "@vrooli/react-component-library/MonetizationAccount/1";
export type UpgradePromptProps = {
  feature: string;
  requiredPlan: string;
  href?: string;
  className?: string;
};
export const UpgradePrompt = withClassName(function UpgradePrompt(props: UpgradePromptProps) {
  return <BaseUpgradePrompt data-testid="monetization.upgrade-prompt" {...props} />;
});
