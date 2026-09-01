import type { Scene } from "./engine";
import { flowCurrent } from "./flowCurrent";
import { hiveLattice } from "./hiveLattice";
import { ledgerRiver } from "./ledgerRiver";
import { orbitalField } from "./orbitalField";
import { panoramaConstellation } from "./panoramaConstellation";
import { signalConstellation } from "./signalConstellation";

/** Compositions are bound by name from the registry; an unknown name gets the orbital field. */
export const compositions: Record<string, () => Scene> = {
  "orbital-field": orbitalField,
  "hive-lattice": hiveLattice,
  "flow-current": flowCurrent,
  "ledger-river": ledgerRiver,
  "signal-constellation": signalConstellation,
  "panorama-constellation": panoramaConstellation,
};

export const createScene = (composition: string): Scene => (compositions[composition] ?? orbitalField)();
