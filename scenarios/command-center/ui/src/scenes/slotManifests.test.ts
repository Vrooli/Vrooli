import { describe, expect, it } from "vitest";
import { compositionSlotManifests } from "./index";
import conversionFunnel from "../../../config/compositions/conversion-funnel.slots.json";
import flowCurrent from "../../../config/compositions/flow-current.slots.json";
import funnelCascade from "../../../config/compositions/funnel-cascade.slots.json";
import hiveLattice from "../../../config/compositions/hive-lattice.slots.json";
import ledgerRiver from "../../../config/compositions/ledger-river.slots.json";
import meridianArc from "../../../config/compositions/meridian-arc.slots.json";
import orbitalField from "../../../config/compositions/orbital-field.slots.json";
import panoramaConstellation from "../../../config/compositions/panorama-constellation.slots.json";
import signalConstellation from "../../../config/compositions/signal-constellation.slots.json";

const generated = {
  "conversion-funnel": conversionFunnel,
  "flow-current": flowCurrent,
  "funnel-cascade": funnelCascade,
  "hive-lattice": hiveLattice,
  "ledger-river": ledgerRiver,
  "meridian-arc": meridianArc,
  "orbital-field": orbitalField,
  "panorama-constellation": panoramaConstellation,
  "signal-constellation": signalConstellation,
};

describe("generated composition slot manifests", () => {
  it("match the module-exported source of truth", () => {
    for (const [composition, slots] of Object.entries(compositionSlotManifests)) {
      expect(generated[composition as keyof typeof generated].composition).toBe(composition);
      expect(generated[composition as keyof typeof generated].slots).toEqual(slots);
    }
  });
});
