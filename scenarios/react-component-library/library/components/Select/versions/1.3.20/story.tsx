// Preview contract exports for Select 1.2.1.
// The declarative story contract owns expectations; this module supplies the
// typed specimen seam for versions whose composition is supplied by a harness.
import { Select } from "./Select";

const options = [
  { value: "ready", label: "Ready" },
  { value: "review", label: "Needs review" },
  { value: "blocked", label: "Blocked", disabled: true },
];

export function Default() {
  return <Select aria-label="Release status" options={options} defaultValue="ready" />;
}
