import { OverlayActionBar, type OverlayActionList } from "./OverlayActionBar";
import {
  fiveGermanOverlayActions,
  missingIDOverlayActions,
  pseudoLocalisedOverlayActions,
} from "../../../../usage-contracts/adversarial-fixtures";

// This story deliberately crosses the type boundary to exercise the runtime warning.
const unsafeFiveGermanActions =
  fiveGermanOverlayActions as unknown as OverlayActionList;
const unsafeMissingIDActions =
  missingIDOverlayActions as unknown as OverlayActionList;
const unsafePseudoLocalisedActions =
  pseudoLocalisedOverlayActions as unknown as OverlayActionList;

export function FiveGermanActions() {
  return (
    <div style={{ width: "min(100vw, 390px)", padding: 12 }}>
      <OverlayActionBar actions={unsafeFiveGermanActions} />
    </div>
  );
}

export function MissingIDs() {
  return (
    <div style={{ width: "min(100vw, 390px)", padding: 12 }}>
      <OverlayActionBar actions={unsafeMissingIDActions} />
    </div>
  );
}

export function PseudoLocalisedLabels() {
  return (
    <div style={{ width: "min(100vw, 390px)", padding: 12 }}>
      <OverlayActionBar actions={unsafePseudoLocalisedActions} />
    </div>
  );
}
