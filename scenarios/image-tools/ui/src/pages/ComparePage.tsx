import { selectors } from "../consts/selectors";
import { CompareView } from "../features/compare/CompareView";

/** Visual-Compare route — drop two images, get a verdict + heat-map + metrics.
 * A Studio surface alongside Workspace, Library, and Smart-Select. */
export function ComparePage() {
  return (
    <div data-testid={selectors.pages.compare}>
      <CompareView />
    </div>
  );
}
