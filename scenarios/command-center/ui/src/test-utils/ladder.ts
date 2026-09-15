import type { Reading } from "../lib/api";
import type { LadderReading } from "../lib/ladder";
import { makeReading } from "./readings";

/** A five-rung schedule in Offer Desk's vocabulary, shaped like the API's ladder reading. */
export const ladderFixture = (overrides: Partial<LadderReading> = {}): LadderReading => ({
  nextRank: 1,
  rungs: [
    { rank: 1, id: "n1", name: "web-console", status: "TRIGGER_MET", finishBar: "CUSTOMER_FACING", ramps: ["desktop"], streams: ["voice_minutes"], audiences: ["developer"],
      blockers: [{ name: "ai-gateway", status: "IDEA", urgency: 1, direct: true }, { name: "scenario-to-desktop", status: "IDEA", urgency: 1 }],
      goals: [{ name: "release-ladder-offer-desk", title: "Offer Desk release-ladder contract", priority: 1 }], readiness: { reported: false } },
    { rank: 2, id: "n2", name: "system-monitor", status: "IDEA", ramps: ["agent"], streams: [], audiences: ["developer"], blockers: [], goals: [], readiness: { reported: false } },
    { rank: 3, id: "n3", name: "git-control-tower", status: "IDEA", ramps: [], streams: [], audiences: ["developer"], blockers: [], goals: [], readiness: { reported: false } },
    { rank: 4, id: "n4", name: "agent-manager", status: "IDEA", ramps: [], streams: ["ai_credits"], audiences: ["business"], blockers: [], goals: [], readiness: { reported: false } },
    { rank: 5, id: "n5", name: "audio-tools", status: "IDEA", ramps: [], streams: [], audiences: ["personal"], blockers: [], goals: [{ name: "audio-reliability-v1", title: "Audio Reliability v1", priority: 5 }], readiness: { reported: false } },
  ],
  enabling: [{ name: "ai-gateway", status: "IDEA", urgency: 1 }, { name: "scenario-to-desktop", status: "IDEA", urgency: 1 }],
  unscheduled: [{ name: "treasury", status: "IDEA" }],
  reach: [
    { kind: "ramp", name: "desktop", opensAt: 1 },
    { kind: "ramp", name: "agent", opensAt: 2 },
    { kind: "ramp", name: "mobile", opensAt: 0 },
    { kind: "stream", name: "voice_minutes", opensAt: 1 },
    { kind: "stream", name: "ai_credits", opensAt: 4 },
    { kind: "audience", name: "developer", opensAt: 1 },
    { kind: "audience", name: "business", opensAt: 4 },
    { kind: "audience", name: "personal", opensAt: 5 },
  ],
  ...overrides,
});

export const ladderReading = (overrides: Partial<Reading> = {}): Reading => makeReading({
  id: "release_ladder", label: "Release ladder", kind: "ladder", value: 5, observedAt: new Date().toISOString(), ttlSeconds: 300,
  source: { team: "director-swarm", binding: "scenario:offer-desk" }, origin_display: "Local instance", ladder: ladderFixture(), ...overrides,
});
