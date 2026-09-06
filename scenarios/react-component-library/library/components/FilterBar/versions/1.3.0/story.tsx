// Preview contract exports for FilterBar 1.1.6.
// The declarative story contract owns expectations; this module supplies the
// typed specimen seam for versions whose composition is supplied by a harness.
export function Default() {
  return null;
}

import { useState } from "react";
import { FilterBar } from "./FilterBar";
export function Compact() {
  const [applied, setApplied] = useState("");
  return (
    <>
      <FilterBar
        density="compact"
        queryLabel="Search conversations"
        queryPlaceholder="Search conversations"
        applyLabel="Search"
        onApply={({ query }) => setApplied(query)}
        onReset={() => setApplied("")}
      />
      <output data-filter-applied>{applied}</output>
    </>
  );
}
