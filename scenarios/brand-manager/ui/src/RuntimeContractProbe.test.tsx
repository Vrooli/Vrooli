import {
  OverlayActionBar,
  type OverlayActionList,
} from "@vrooli/react-component-library/OverlayActionBar/1";

// Deliberately adversarial data for EX7's isolated served-runtime probe.
const actions: OverlayActionList = [
  {
    id: "primary",
    label: "Vollständige Wiederherstellung der archivierten Sitzung starten",
    primary: true,
    onSelect: () => undefined,
  },
  { id: "two", label: "Aktion zwei", onSelect: () => undefined },
  { id: "three", label: "Aktion drei", onSelect: () => undefined },
  {
    id: "four",
    label: "Dauerhaften Kontinuitätsbeleg mit allen Details anzeigen",
    onSelect: () => undefined,
  },
  {
    id: "five",
    label: "Vollständigen Gesprächsverlauf sicher exportieren",
    onSelect: () => undefined,
  },
];

export function RuntimeContractProbe() {
  return (
    <div className="fixed inset-x-4 bottom-4 z-50 rounded bg-white p-3 shadow">
      <OverlayActionBar actions={actions} />
    </div>
  );
}
