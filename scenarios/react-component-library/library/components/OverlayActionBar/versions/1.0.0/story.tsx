import { OverlayActionBar } from "./OverlayActionBar";
import { fiveGermanOverlayActions, missingIDOverlayActions, pseudoLocalisedOverlayActions } from "../../../../usage-contracts/adversarial-fixtures";

export function FiveGermanActions() {
  return <div style={{ width: "min(100vw, 390px)", padding: 12 }}><OverlayActionBar actions={fiveGermanOverlayActions} /></div>;
}

export function MissingIDs() {
  return <div style={{ width: "min(100vw, 390px)", padding: 12 }}><OverlayActionBar actions={missingIDOverlayActions} /></div>;
}

export function PseudoLocalisedLabels() {
  return <div style={{ width: "min(100vw, 390px)", padding: 12 }}><OverlayActionBar actions={pseudoLocalisedOverlayActions} /></div>;
}
