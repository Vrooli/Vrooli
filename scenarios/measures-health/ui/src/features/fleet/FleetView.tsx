import { useState } from "react";

import { FleetTable } from "./FleetTable";
import { ScenarioCoverageCard } from "./ScenarioCoverageCard";

/**
 * FleetView composes the cross-scenario coverage table with the per-scenario
 * drill-down, owning the selection state between them. Selecting a row in the
 * table loads that scenario's domain-level coverage in the card beneath it.
 */
export function FleetView() {
  const [selected, setSelected] = useState<string | undefined>(undefined);

  return (
    <div className="flex flex-col gap-4">
      <FleetTable selected={selected} onSelect={setSelected} />
      <ScenarioCoverageCard scenario={selected} />
    </div>
  );
}
