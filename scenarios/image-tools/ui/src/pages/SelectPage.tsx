import { selectors } from "../consts/selectors";
import { SmartSelectView } from "../features/select/SmartSelectView";

/** Smart-Select route — click a region, classify it, and apply a context-aware
 * edit. A Studio surface alongside Workspace + Library. */
export function SelectPage() {
  return (
    <div data-testid={selectors.pages.select}>
      <SmartSelectView />
    </div>
  );
}
