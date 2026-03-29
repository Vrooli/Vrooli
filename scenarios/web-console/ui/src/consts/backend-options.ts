// CROSS-LANGUAGE COUPLING: Backend IDs must match BackendID constants in api/backend_registry.go
import type { BackendID } from "../lib/api";

export interface BackendOptionConst {
  id: BackendID;
  label: string;
  description: string;
  survivesRestart: boolean;
}

export const BACKEND_OPTIONS: BackendOptionConst[] = [
  {
    id: "standard",
    label: "Standard",
    description: "Lightweight session. Lost if web console restarts.",
    survivesRestart: false,
  },
  {
    id: "persistent",
    label: "Persistent",
    description: "Survives restarts. Ideal for long-running tasks.",
    survivesRestart: true,
  },
];
