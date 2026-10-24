import { spawnSync } from "node:child_process";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

const apiBase = process.env.PLANNER_PARITY_API_BASE;
const eventId = process.env.PLANNER_PARITY_EVENT_ID;
const cliBinary = process.env.PLANNER_PARITY_CLI;
const actorToken = process.env.PLANNER_PARITY_ACTOR_TOKEN;

describe.skipIf(!apiBase || !eventId || !cliBinary || !actorToken)("Planner UI and CLI parity against routed test storage", () => {
  let calendar: typeof import("./calendar");
  let workspace: typeof import("./workspace");

  beforeAll(async () => {
    const host = new URL(apiBase!);
    const windowOverride = Object.create(globalThis.window) as Window;
    Object.defineProperty(windowOverride, "location", {
      value: { origin: apiBase, hostname: host.hostname, pathname: "/" },
    });
    vi.stubGlobal("window", windowOverride);

    const fetchImpl = globalThis.fetch.bind(globalThis);
    vi.stubGlobal("fetch", (input: RequestInfo | URL, init?: RequestInit) => {
      const headers = new Headers(input instanceof Request ? input.headers : undefined);
      new Headers(init?.headers).forEach((value, name) => headers.set(name, value));
      headers.set("X-Vrooli-Test-Mode", "1");
      headers.set("Authorization", `Bearer ${actorToken}`);
      return fetchImpl(input, { ...init, headers });
    });

    calendar = await import("./calendar");
    workspace = await import("./workspace");
  });

  afterAll(() => vi.unstubAllGlobals());

  it("shares event and profile identity/revisions through UI REST, UI Connect and the real CLI", async () => {
    const initialEvent = await calendar.fetchCalendarEvent(eventId!);
    expect(initialEvent).toMatchObject({ id: eventId, title: "CLI updated routed event", revision: 3 });

    const connectEvent = await calendar.calendarClient.getEvent({ eventId: eventId! });
    expect(connectEvent.event).toMatchObject({ id: eventId, title: initialEvent.title, revision: 3n });

    const uiUpdatedEvent = await calendar.updateCalendarEvent(eventId!, {
      title: "UI TypeScript client event",
      subject: initialEvent.subject,
      notes: initialEvent.notes,
      availability: "busy",
      timezone: initialEvent.timezone,
      all_day: initialEvent.all_day,
      start_date: initialEvent.start_date,
      end_date_exclusive: initialEvent.end_date_exclusive,
      start_at: initialEvent.start_at,
      end_at: initialEvent.end_at,
    }, initialEvent.revision);
    expect(uiUpdatedEvent).toMatchObject({ id: eventId, title: "UI TypeScript client event", revision: 4 });

    const connectReadback = await calendar.calendarClient.getEvent({ eventId: eventId! });
    expect(connectReadback.event).toMatchObject({ id: eventId, title: uiUpdatedEvent.title, revision: 4n });
    expect(runCLI("calendar", "get-event", "--event-id", eventId!)).toContain("revision 4");

    const initialProfile = await workspace.fetchPlanningProfile();
    expect(initialProfile).toMatchObject({ id: "default", timezone: "America/New_York", revision: 2n });
    expect(runCLI("workspace", "profile")).toContain("revision=2");

    const uiUpdatedProfile = await workspace.updatePlanningProfile(initialProfile, {
      timezone: "America/New_York",
      weekStart: "sunday",
      dailyCapacityMinutes: 400,
      reserveMinutes: 40,
      focusSessionMinutes: 35,
    });
    expect(uiUpdatedProfile).toMatchObject({ id: "default", weekStart: "sunday", revision: 3n });
    expect(await workspace.fetchPlanningProfile()).toMatchObject({ id: "default", weekStart: "sunday", revision: 3n });
    expect(runCLI("workspace", "profile")).toContain("revision=3");
  });
});

function runCLI(...args: string[]): string {
  const env = { ...process.env };
  delete env.VROOLI_API_TOKEN;
  delete env.PERSONAL_PLANNER_TEST_MODE;
  delete env.PERSONAL_PLANNER_API_BASE;
  delete env.PERSONAL_PLANNER_API_URL;
  delete env.PERSONAL_PLANNER_API_PORT;
  delete env.PERSONAL_PLANNER_CONFIG_DIR;
  delete env.VROOLI_CLI_CONFIG_DIR;
  delete env.API_BASE_URL;
  delete env.VITE_API_BASE_URL;
  const result = spawnSync(cliBinary!, ["--api-base", apiBase!, ...args], {
    encoding: "utf8",
    env: {
      ...env,
      PERSONAL_PLANNER_API_BASE: apiBase!,
      PERSONAL_PLANNER_TEST_MODE: "1",
      VROOLI_API_TOKEN: actorToken!,
    },
  });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`Planner CLI ${args.join(" ")} failed: ${result.stdout}\n${result.stderr}`);
  return `${result.stdout}\n${result.stderr}`;
}
