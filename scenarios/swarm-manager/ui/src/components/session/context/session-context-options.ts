import type { AgentSessionContextRef } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";

export function optionsToRefs(options: SessionContextOption[]): AgentSessionContextRef[] {
  return options.map(({ type, ref }) => ({ type, ref }));
}
