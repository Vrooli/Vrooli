/**
 * The reading screens, rendered from wire-shaped replies: every derived value
 * shows its source one tap away, unknown stays unknown, and nothing is written
 * except by the one "Mark as seen" action.
 */
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders, stubWire } from "../test-utils";

const ACTIVITY = "/vrooli.research_desk.v1.activity.ActivityService";
const OPERATOR = "/vrooli.research_desk.v1.operator.OperatorService";
const LAUNCH = "/vrooli.research_desk.v1.launch.LaunchService";
const HOME = "/home/u/.vrooli/research";
const AT = "2026-11-11T09:30:00Z";

/** Values the API sends; each must reach the screen verbatim. */
const WIRE = {
  stateRule: "goal active in an open window",
  stateReason: "goal status active",
  claim: "IT6 landed",
  session: "b7117dc2",
  step: "session-created",
  showcase: "Qwen 3 fits in 24 GB",
  rate: "41 tok/s",
  flag: "charter is over its 24 KB ceiling",
  otherTopic: "other-topic",
  mode: "delivery",
  workstream: "IT6",
  missionError: "missing showcase",
  missionWarning: "long card",
  title: "Programme one",
  answeredTitle: "Answered",
  lensReason: "connection refused",
  startCommand: "vrooli scenario start notification-hub",
  writerReason: "another desk is the writer",
  automaticReason: "web-console down",
  bareCapability: "pm",
  verdict: "Keep going.",
  pausedRule: "supervisor window open; goal paused; allowance ended",
  atLeast: "at least: the run-folder walk stopped at its limit",
  unreadable: "cannot read STATE.md: permission denied",
  courierHealth: "last pass 01:55Z: no switch on, nothing is sent; last delivery never",
  baseline: "no look recorded; the last 24 hours",
  lastDay: "no look recorded; the last 24 hours since 2026-11-10T09:30Z",
  writtenWhilePaused: "written while paused: an owner may still be running",
  touched: "README.md edited after the end",
  access: "local availability policy (no desk credential)",
  newerSource: "~research/topics/newer/runs/c1/notes.md",
  accessLens: "Web Console access: local availability policy (no desk credential)",
} as const;

const render = (path: string) => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

afterEach(() => cleanup());

const commission = {
  id: "c1",
  folder: `${HOME}/cohorts/c1`,
  title: { value: WIRE.title },
  state: "COMMISSION_STATE_LIVE",
  stateRule: WIRE.stateRule,
  stateEvidence: [{ known: true, source: `${HOME}/cohorts/c1/LAUNCH.json`, reason: WIRE.stateReason, asOf: AT }],
  topics: [{ topic: "local-model-performance", basis: "charter" }],
  topicsEvidence: { known: true, source: "CHARTER.md" },
  started: { value: AT, evidence: { known: true } },
  allowanceEnd: { evidence: { known: false, reason: "no allowance line in the charter" } },
  lastWrite: { value: AT, evidence: { known: true } },
  claim: { value: WIRE.claim, evidence: { known: true, source: `${HOME}/cohorts/c1/STATE.md` } },
};

const detail = {
  commission,
  matches: [{ kind: "MATCH_KIND_EXACT", basis: "launch-record", webConsoleSessionId: WIRE.session, goalStatus: "active", at: AT, windowOpen: true, source: `${HOME}/cohorts/c1/LAUNCH.json` }],
  files: [{ name: "STATE.md", path: `${HOME}/cohorts/c1/STATE.md`, bytes: "4200", modified: AT }],
  runs: [
    { topic: "local-model-performance", path: `${HOME}/topics/local-model-performance/runs/c1`, lastWrite: { value: AT }, checkpoint: { name: "CHECKPOINT.md", path: `${HOME}/topics/local-model-performance/runs/c1/CHECKPOINT.md`, bytes: "900" } },
    { topic: WIRE.otherTopic, path: `${HOME}/topics/other-topic/runs/c1` },
  ],
  readSets: [
    { role: "scheduler-start", bytes: "36000", files: [{ name: "CHARTER.md", path: `${HOME}/cohorts/c1/CHARTER.md`, bytes: "30000" }], flags: [WIRE.flag] },
    { role: "owner-start:local-model-performance", bytes: "8000", files: [], flags: [] },
  ],
  ceilings: { known: true, source: "limits.json" },
};

const changes = {
  since: AT,
  ledger: [{ path: `${HOME}/cohorts/c1/LEDGER.md`, text: `- IT6 measured at **${WIRE.rate}**`, truncated: true }],
  checkpoints: [{ name: "CHECKPOINT.md", path: `${HOME}/topics/local-model-performance/runs/c1/CHECKPOINT.md`, bytes: "900" }],
  showcaseLines: [WIRE.showcase],
  statusChanges: [{ topic: "local-model-performance", workstream: WIRE.workstream, before: "active", after: "landed" }],
  files: [{ name: "notes.md", path: `${HOME}/topics/local-model-performance/notes.md`, bytes: "120", modified: AT }],
  truncated: true,
};

function wire(overrides: Record<string, unknown> = {}) {
  const replies: Record<string, unknown> = {
    [`${ACTIVITY}/GetOverview`]: { researchHome: HOME },
    [`${ACTIVITY}/GetCommission`]: detail,
    [`${ACTIVITY}/GetChanges`]: changes,
    [`${LAUNCH}/GetLaunch`]: {
      launch: {
        commissionId: "c1",
        path: `${HOME}/cohorts/c1/LAUNCH.json`,
        steps: [{ step: WIRE.step, at: AT }],
        controls: [{ action: "pause", at: AT }],
        failedStep: "goal-set",
        failedReason: "goal refused",
        discardedAt: AT,
        discardedReason: "wrong topics",
      },
    },
    ...overrides,
  };
  return stubWire((path) => replies[path] ?? {});
}

describe("Commission status", () => {
  it("shows each derived value with its evidence a tap away, and unknown with its reason", async () => {
    const user = userEvent.setup();
    wire();
    render("/commissions/c1");
    expect(await screen.findByText(WIRE.stateRule)).toBeInTheDocument();
    const why = screen.getAllByRole("button", { name: strings.evidence.why })[0];
    expect(why).toHaveAttribute("aria-expanded", "false");
    await user.click(why as HTMLElement);
    expect(screen.getByText(WIRE.stateReason)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.evidence.hide })).toHaveAttribute("aria-expanded", "true");
    // The allowance could not be derived: it reads unknown, never a guess.
    expect(screen.getAllByText(strings.common.unknown).length).toBeGreaterThan(0);
    expect(screen.getByText(WIRE.claim).tagName).toBe("Q");
    expect(screen.getByRole("link", { name: "local-model-performance" })).toHaveAttribute("href", "/topics/local-model-performance");
  });

  it("names the matched session and the launch record's steps, failure and discard", async () => {
    wire();
    render("/commissions/c1");
    expect(await screen.findByText(WIRE.session)).toBeInTheDocument();
    expect(screen.getByText(new RegExp(strings.commission.exactMatch))).toBeInTheDocument();
    expect(await screen.findByText(WIRE.step)).toBeInTheDocument();
    expect(screen.getByText(strings.commission.controlEntry)).toBeInTheDocument();
    expect(screen.getByText(strings.launch.failedAt)).toBeInTheDocument();
    expect(screen.getByText(strings.commission.discarded)).toBeInTheDocument();
  });

  it("shows an older layout with nothing derivable as unknown throughout, without error", async () => {
    wire({ [`${ACTIVITY}/GetCommission`]: { commission: { id: "old", folder: `${HOME}/cohorts/old`, state: "COMMISSION_STATE_UNKNOWN" } }, [`${LAUNCH}/GetLaunch`]: {} });
    render("/commissions/old");
    expect(await screen.findByText(strings.commission.byHand)).toBeInTheDocument();
    // state rule, topics, started, allowance end, last write and claim.
    expect(screen.getAllByText(strings.common.unknown).length).toBeGreaterThanOrEqual(6);
    expect(screen.queryByText(strings.commission.launchRecord)).not.toBeInTheDocument();
  });

  it("says unknown when no session can be matched", async () => {
    wire({ [`${ACTIVITY}/GetCommission`]: { ...detail, matches: [] } });
    render("/commissions/c1");
    expect(await screen.findByText(new RegExp(strings.commission.noSession))).toBeInTheDocument();
  });
});

describe("What changed", () => {
  it("lists ledger text, statuses and files, and records a look only on Mark as seen", async () => {
    const user = userEvent.setup();
    const calls = wire();
    render("/commissions/c1");
    expect(await screen.findByText(strings.changes.statusLine)).toBeInTheDocument();
    expect(screen.getByText(WIRE.showcase)).toBeInTheDocument();
    expect(screen.getByText(WIRE.rate).tagName).toBe("STRONG");
    expect(screen.getByText(strings.changes.truncatedText)).toBeInTheDocument();
    expect(screen.getByText(strings.changes.truncatedWalk)).toBeInTheDocument();
    expect(calls.some((c) => c.path.endsWith("/MarkSeen"))).toBe(false);
    await user.click(screen.getByRole("button", { name: strings.changes.markSeen }));
    await waitFor(() => expect(calls.filter((c) => c.path.endsWith("/MarkSeen"))).toEqual([{ path: `${OPERATOR}/MarkSeen`, body: { subject: "CHANGE_SUBJECT_COMMISSION", id: "c1" } }]));
  });

  it("with no look recorded reports the last 24 hours and says so, and says when nothing changed (rulings P7)", async () => {
    wire({ [`${ACTIVITY}/GetChanges`]: { since: AT, baseline: WIRE.lastDay, baselineKind: "BASELINE_LAST_DAY" } });
    render("/commissions/c1");
    expect(await screen.findByText(WIRE.lastDay)).toBeInTheDocument();
    expect(screen.getByText(strings.changes.nothing)).toBeInTheDocument();
    cleanup();
    wire({ [`${ACTIVITY}/GetChanges`]: { since: AT } });
    render("/commissions/c1");
    expect(await screen.findByText(strings.changes.nothing)).toBeInTheDocument();
  });

  it("shows the API's own comparison line, and a list row says the last 24 hours when no look was recorded", async () => {
    wire({
      [`${ACTIVITY}/GetChanges`]: { since: AT, baseline: WIRE.baseline, baselineKind: "BASELINE_LAST_DAY", files: [{ path: `${HOME}/cohorts/c1/STATE.md`, bytes: "10" }] },
      [`${ACTIVITY}/GetOverview`]: { researchHome: HOME, stateCounts: [{ state: "COMMISSION_STATE_LIVE", count: 1 }], running: [commission] },
    });
    render("/commissions/c1");
    expect(await screen.findByText(WIRE.baseline)).toBeInTheDocument();
    expect(screen.queryByText(strings.changes.nothing)).not.toBeInTheDocument();
    cleanup();
    render("/");
    expect(await screen.findByText(strings.changes.countDay)).toBeInTheDocument();
  });

  it("compares with a time the operator picks, sending it as since", async () => {
    const user = userEvent.setup();
    const calls = wire();
    render("/commissions/c1");
    await screen.findByText(strings.changes.statusLine);
    expect(calls.filter((c) => c.path.endsWith("/GetChanges")).every((c) => !("since" in (c.body as object)))).toBe(true);
    await user.click(screen.getByText(strings.changes.sinceTime));
    const from = screen.getByLabelText(strings.changes.from);
    await user.clear(from);
    await user.type(from, "2026-11-10T08:00");
    await waitFor(() =>
      expect(calls.filter((c) => c.path.endsWith("/GetChanges")).map((c) => (c.body as { since?: string }).since)).toContain(new Date("2026-11-10T08:00").toISOString().replace(".000Z", "Z")),
    );
  });

  it("lists files it cannot read as unknown instead of reporting nothing changed", async () => {
    wire({ [`${ACTIVITY}/GetChanges`]: { since: AT, unreadable: [{ source: `${HOME}/cohorts/c1/LEDGER.md`, reason: WIRE.unreadable }] } });
    render("/commissions/c1");
    expect(await screen.findByText(WIRE.unreadable)).toBeInTheDocument();
    expect(screen.queryByText(strings.changes.nothing)).not.toBeInTheDocument();
  });
});

describe("Commission reading and files", () => {
  it("shows what each agent reads to start, with the API's flags", async () => {
    wire();
    render("/commissions/c1?tab=reading");
    expect(await screen.findByText(strings.reading.schedulerStart)).toBeInTheDocument();
    expect(screen.getByText(strings.reading.ownerStart)).toBeInTheDocument();
    expect(screen.getByText(WIRE.flag)).toBeInTheDocument();
    expect(screen.getByText(strings.reading.ceilingsKnown)).toBeInTheDocument();
  });

  it("says when there is nothing to measure", async () => {
    wire({ [`${ACTIVITY}/GetCommission`]: { ...detail, readSets: [] } });
    render("/commissions/c1?tab=reading");
    expect(await screen.findByText(strings.reading.none)).toBeInTheDocument();
  });

  it("lists the folder and each run with its checkpoint, or says it has none", async () => {
    wire();
    render("/commissions/c1?tab=files");
    const runs = await screen.findByRole("region", { name: strings.files.runs });
    expect(within(runs).getByText(WIRE.otherTopic)).toBeInTheDocument();
    expect(within(runs).getByText(strings.files.noCheckpoint)).toBeInTheDocument();
    expect(within(runs).getByText(new RegExp(strings.files.runChanged))).toBeInTheDocument();
  });
});

const topic = (name: string, seconds: string, extra: Record<string, unknown> = {}) => ({
  topic: name,
  title: { value: `Title ${name}` },
  lastResearchWrite: { topic: name, commissionId: "c1", at: { value: seconds, evidence: { known: true, source: `${HOME}/topics/${name}/runs/c1/notes.md` } } },
  commissionIds: ["c1"],
  liveCommissionIds: [],
  pausedCommissionIds: [],
  limitedCommissionIds: [],
  ...extra,
});

describe("Topics", () => {
  it("lists the newest research write first, with live, paused and limited apart, the write's source a tap away, and a filter", async () => {
    wire({
      [`${ACTIVITY}/ListTopics`]: {
        topics: [
          topic("older", "2026-10-01T00:00:00Z", {
            pausedCommissionIds: ["c2"],
            lastResearchWrite: { topic: "older", commissionId: "c2", note: WIRE.writtenWhilePaused, at: { value: "2026-10-01T00:00:00Z", evidence: { known: true } } },
          }),
          topic("newer", AT, {
            liveCommissionIds: ["c1"],
            limitedCommissionIds: ["c3"],
            mission: { path: `${HOME}/topics/newer/MISSION.md`, bytes: "8000" },
            missionCheck: { mode: WIRE.mode, open: 2, active: 1, warnings: ["no showcase"] },
          }),
          { topic: "silent", commissionIds: [], liveCommissionIds: [], pausedCommissionIds: [], limitedCommissionIds: [] },
        ],
      },
    });
    render("/topics");
    const links = await screen.findAllByRole("link", { name: /Title|silent/ });
    expect(links.map((l) => l.getAttribute("href"))).toEqual(["/topics/newer", "/topics/older", "/topics/silent"]);
    expect(screen.getByText(WIRE.mode)).toBeInTheDocument();
    // A write under a paused owner says so (rulings P8).
    expect(screen.getByText(WIRE.writtenWhilePaused)).toBeInTheDocument();
    expect(screen.getByText(strings.topics.problems)).toBeInTheDocument();
    expect(screen.getAllByText(strings.topics.noCheck, { exact: false })).toHaveLength(2);
    const user = userEvent.setup();
    await user.click(screen.getAllByRole("button", { name: strings.evidence.why })[0] as HTMLElement);
    expect(screen.getByRole("link", { name: WIRE.newerSource })).toBeInTheDocument();
    await user.type(screen.getByRole("searchbox", { name: strings.topics.filter }), "OLD");
    expect(screen.getAllByRole("link", { name: /Title|silent/ }).map((l) => l.getAttribute("href"))).toEqual(["/topics/older"]);
    await user.clear(screen.getByRole("searchbox", { name: strings.topics.filter }));
    await user.type(screen.getByRole("searchbox", { name: strings.topics.filter }), "nothing");
    expect(screen.getByText(strings.topics.noMatch)).toBeInTheDocument();
  });

  it("says so when the home has no topics", async () => {
    wire({ [`${ACTIVITY}/ListTopics`]: { topics: [] } });
    render("/topics");
    expect(await screen.findByText(strings.topics.emptyTitle)).toBeInTheDocument();
  });

  it("shows one topic's mission check, workstreams, commissions and changes", async () => {
    wire({
      [`${ACTIVITY}/ListTopics`]: {
        topics: [
          topic("t", AT, {
            mission: { path: `${HOME}/topics/t/MISSION.md`, bytes: "8000" },
            missionCheck: { mode: "research", errors: [WIRE.missionError], warnings: [WIRE.missionWarning], workstreams: [{ id: WIRE.workstream, status: "active", showcase: "41 tok/s" }, { id: "IT7", status: "landed" }] },
          }),
        ],
      },
      [`${ACTIVITY}/ListCommissions`]: { commissions: [commission] },
    });
    render("/topics/t");
    expect(await screen.findByText(WIRE.workstream)).toBeInTheDocument();
    expect(screen.getByText(WIRE.missionError)).toBeInTheDocument();
    expect(screen.getByText(WIRE.missionWarning)).toBeInTheDocument();
    expect(await screen.findByText(WIRE.title)).toBeInTheDocument();
  });

  it("names a topic that is not in the home", async () => {
    wire({ [`${ACTIVITY}/ListTopics`]: { topics: [] } });
    render("/topics/gone");
    expect(await screen.findByText(strings.topics.missing)).toBeInTheDocument();
  });
});

describe("Commissions", () => {
  it("groups by state, newest write first, with ended ones folded", async () => {
    wire({
      [`${ACTIVITY}/ListCommissions`]: {
        commissions: [
          { ...commission, id: "older-live", title: { value: "Older live" }, lastWrite: { value: "2026-11-01T00:00:00Z" } },
          commission,
          { id: "done", title: { value: "Finished" }, state: "COMMISSION_STATE_ENDED" },
          { id: "odd", title: { value: "Unplaced" } },
        ],
      },
    });
    const { container } = render("/commissions");
    const live = await screen.findByRole("region", { name: strings.state.live });
    expect(within(live).getAllByRole("link", { name: /live|one/i }).filter((l) => l.classList.contains("rd-row-link")).map((l) => l.getAttribute("href"))).toEqual(["/commissions/c1", "/commissions/older-live"]);
    expect([...container.querySelectorAll("details.rd-fold")].map((d) => d.textContent)).toEqual([expect.stringContaining("Unplaced"), expect.stringContaining("Finished")]);
  });

  it("opens a folded group when the link asks for that state", async () => {
    wire({ [`${ACTIVITY}/ListCommissions`]: { commissions: [{ id: "odd", title: { value: "Unplaced" } }] } });
    render("/commissions?state=3");
    expect(await screen.findByRole("region", { name: strings.state.unknown })).toBeInTheDocument();
  });

  it("shows an edit after the end as touched after end, beside the last write inside the interval", async () => {
    wire({
      [`${ACTIVITY}/ListCommissions`]: {
        commissions: [{ ...commission, id: "done", state: "COMMISSION_STATE_ENDED", ended: { value: AT, evidence: { known: true } }, touchedAfterEnd: { value: AT, evidence: { known: true, reason: WIRE.touched } } }],
      },
      [`${ACTIVITY}/GetCommission`]: {
        ...detail,
        commission: { ...commission, state: "COMMISSION_STATE_ENDED", ended: { value: AT, evidence: { known: true } }, touchedAfterEnd: { value: AT, evidence: { known: true, reason: WIRE.touched } } },
        unreadable: [{ source: `${HOME}/cohorts/c1/OPERATOR.md`, reason: WIRE.unreadable }],
      },
    });
    render("/commissions?state=4");
    expect(await screen.findByText(strings.commission.touchedAfterEnd, { exact: false })).toBeInTheDocument();
    cleanup();
    render("/commissions/c1");
    expect(await screen.findByText(strings.commission.touchedAfterEndLabel)).toBeInTheDocument();
    expect(screen.getByText(strings.commission.ended)).toBeInTheDocument();
    expect(screen.getByText(strings.commission.touchedNote, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(WIRE.unreadable)).toBeInTheDocument();
  });

  it("says so when the home has no commissions", async () => {
    wire({ [`${ACTIVITY}/ListCommissions`]: { commissions: [] } });
    render("/commissions");
    expect(await screen.findByText(strings.commissions.emptyTitle)).toBeInTheDocument();
  });
});

describe("Reviews", () => {
  it("lists review sessions, or says there are none", async () => {
    wire({ [`${OPERATOR}/ListReviews`]: { reviews: [{ id: "r1", title: { value: "Review one" }, topics: ["t"], verdicts: [] }] } });
    render("/reviews");
    expect(await screen.findByRole("link", { name: /Review one/ })).toHaveAttribute("href", "/reviews/r1");
    cleanup();
    wire({ [`${OPERATOR}/ListReviews`]: { reviews: [] } });
    render("/reviews");
    expect(await screen.findByText(strings.reviews.emptyTitle)).toBeInTheDocument();
  });
});

describe("Files", () => {
  it("renders a Markdown file with its links resolved against its folder", async () => {
    const calls = wire({ "/api/v1/home/cohorts/c1/STATE.md": "# State\nSee [ledger](LEDGER.md)." });
    render("/files/cohorts/c1/STATE.md");
    expect(await screen.findByRole("link", { name: "ledger" })).toHaveAttribute("href", "/files/cohorts/c1/LEDGER.md");
    expect(calls.some((c) => c.path === "/api/v1/home/cohorts/c1/STATE.md")).toBe(true);
  });

  it("opens HTML in its own tab instead of framing it, and shows images inline", async () => {
    wire();
    render("/files/topics/t/REVIEW.html");
    const open = (await screen.findAllByRole("link", { name: /REVIEW\.html/ })).find((l) => l.getAttribute("target") === "_blank");
    expect(open?.getAttribute("href")).toContain("/api/v1/home/topics/t/REVIEW.html");
    cleanup();
    wire();
    render("/files/topics/t/chart.png");
    expect(await screen.findByRole("img", { name: "chart.png" })).toBeInTheDocument();
  });

  it("refuses a path outside the research home", async () => {
    wire();
    render("/files/..%2Fsecrets");
    expect(await screen.findByText(strings.files.outside)).toBeInTheDocument();
  });
});

describe("Now", () => {
  const needs = (courier: Record<string, unknown>) => ({
    items: [
      { id: "AV4", title: "Choose how AV4 handles yt-dlp", commissionId: "c1" },
      { id: "G1", title: "Global ask" },
      { id: "old", title: WIRE.answeredTitle, commissionId: "c1", answer: { text: "keep" } },
    ],
    courier,
  });

  const paused = (id: string, extra: Record<string, unknown> = {}) => ({
    ...commission,
    id,
    title: { value: id },
    state: "COMMISSION_STATE_PAUSED",
    stateRule: WIRE.pausedRule,
    allowanceEnd: { value: AT, evidence: { known: true } },
    ...extra,
  });

  it("answers what is moving first: status line, then what needs the operator, then supervisors with topics (rulings V5)", async () => {
    const errors = vi.spyOn(console, "error");
    wire({
      [`${ACTIVITY}/GetOverview`]: {
        researchHome: HOME,
        generatedAt: AT,
        stateCounts: [{ state: "COMMISSION_STATE_LIVE", count: 1 }, { state: "COMMISSION_STATE_PAUSED", count: 1 }],
        running: [commission, paused("c2")],
        // One overlap per pair per topic, as the API sends it; c3 is history (not open now).
        overlaps: [
          { topic: "local-model-performance", commissionIds: ["c1", "c2"] },
          { topic: WIRE.otherTopic, commissionIds: ["c2", "c1"] },
          { topic: WIRE.otherTopic, commissionIds: ["c1", "c3"] },
        ],
        lenses: [{ name: "agent-manager", available: false, reason: WIRE.lensReason }],
        unreadable: [{ source: `${HOME}/cohorts/c9/STATE.md`, reason: WIRE.unreadable }],
        latestResearchWrite: { topic: "newer", commissionId: "c1", at: { value: AT, evidence: { known: true } } },
      },
      [`${OPERATOR}/ListNeedsYou`]: needs({ writer: true, globalOn: true, hubRead: true }),
      [`${ACTIVITY}/ListTopics`]: { topics: [topic("older", "2026-10-01T00:00:00Z"), topic("newer", AT)] },
      [`${ACTIVITY}/ListCommissions`]: { commissions: [{ id: "p1", title: { value: "Packet one" }, state: "COMMISSION_STATE_PREPARED" }] },
    });
    render("/");
    const status = await screen.findByTestId(selectors.now.status);
    expect(within(status).getByText(strings.now.openSummary)).toBeInTheDocument();
    expect(await within(status).findByRole("link", { name: "newer" })).toHaveAttribute("href", "/topics/newer");
    // Items link to commission plus ID, never the bare ID (V2).
    expect(await screen.findByRole("link", { name: /Choose how AV4 handles yt-dlp/ })).toHaveAttribute("href", "/needs-you#c1%2FAV4");
    expect(screen.getByRole("link", { name: /Global ask/ })).toHaveAttribute("href", "/needs-you#global%2FG1");
    expect(screen.queryByText(WIRE.answeredTitle)).not.toBeInTheDocument();
    const needsYou = screen.getByRole("heading", { name: strings.now.needsYou });
    const live = await screen.findByRole("heading", { name: strings.now.liveTitle });
    expect(status.compareDocumentPosition(needsYou) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(needsYou.compareDocumentPosition(live) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByRole("heading", { name: strings.now.pausedTitle })).toBeInTheDocument();
    expect(screen.getAllByText(strings.state.pastAllowance)).toHaveLength(1);
    const counts = screen.getByRole("list", { name: strings.now.stateCounts });
    expect(within(counts).getAllByRole("link").map((l) => l.getAttribute("href"))).toEqual(["/commissions?state=5", "/commissions?state=6"]);
    expect(screen.getByText(WIRE.lensReason)).toBeInTheDocument();
    expect(screen.getByText(WIRE.unreadable)).toBeInTheDocument();
    // Only pairs of commissions open now, each once, with every topic they share.
    const pairs = screen.getAllByRole("list", { name: strings.now.overlapPair });
    expect(pairs.map((pair) => within(pair).getAllByRole("link").map((l) => l.textContent))).toEqual([["c1", "c2"]]);
    expect(within(screen.getByRole("list", { name: strings.now.overlapTopics })).getAllByRole("link")).toHaveLength(2);
    expect(errors.mock.calls.filter((call) => /same key/.test(String(call[0])))).toEqual([]);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: strings.launch.launchPrepared }));
    expect(await screen.findByTestId(selectors.pages.launchStatus)).toBeInTheDocument();
  });

  it("reads 'nothing live, two paused' as a state, with each saying its allowance ended", async () => {
    wire({
      [`${ACTIVITY}/GetOverview`]: {
        researchHome: HOME,
        stateCounts: [{ state: "COMMISSION_STATE_PAUSED", count: 2 }, { state: "COMMISSION_STATE_UNKNOWN", count: 18 }],
        running: [paused("c1", { lastWrite: { value: AT, evidence: { known: true, reason: WIRE.atLeast } } }), paused("c2")],
      },
      [`${OPERATOR}/ListNeedsYou`]: { items: [], courier: { writer: true, globalOn: true, hubRead: true } },
    });
    render("/");
    expect(await screen.findByText(strings.now.openSummary)).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: strings.now.liveTitle })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: strings.now.pausedTitle })).toBeInTheDocument();
    expect(screen.getByText(strings.now.pausedNote)).toBeInTheDocument();
    expect(screen.getAllByText(strings.state.pastAllowance)).toHaveLength(2);
    // A cut-short walk is a lower bound, said as such (F3).
    expect(screen.getAllByText(strings.common.atLeast, { exact: false })).toHaveLength(1);
    expect(screen.queryByText(strings.now.nothingOpen)).not.toBeInTheDocument();
  });

  it("says when no window is open and nothing needs the operator", async () => {
    wire({ [`${OPERATOR}/ListNeedsYou`]: { items: [], courier: { writer: true, globalOn: true, hubRead: true } } });
    render("/");
    expect(await screen.findByText(strings.now.nothingNeedsYou)).toBeInTheDocument();
    expect(await screen.findByText(strings.now.nothingOpen)).toBeInTheDocument();
  });

  it("shows what changed since the last look or a chosen period, and marks everything seen only after confirming (rulings V7)", async () => {
    const user = userEvent.setup();
    const recent = new Date(Date.now() - 3_600_000).toISOString();
    const calls = wire({
      [`${ACTIVITY}/ListTopics`]: { topics: [topic("newer", recent), topic("older", "2026-10-01T00:00:00Z")] },
      [`${ACTIVITY}/GetChanges`]: { since: AT, baselineKind: "BASELINE_LOOK", files: [{ path: `${HOME}/topics/newer/runs/c1/notes.md`, bytes: "10" }] },
      [`${OPERATOR}/MarkSeen`]: { at: AT, commissions: 24, topics: 27 },
    });
    render("/");
    const since = await screen.findByRole("region", { name: strings.now.since });
    // Only topics written in the period are listed, each with its changes since the look.
    expect(await within(since).findByRole("link", { name: "Title newer" })).toBeInTheDocument();
    expect(within(since).queryByRole("link", { name: "Title older" })).not.toBeInTheDocument();
    expect(await within(since).findByText(strings.changes.countSince)).toBeInTheDocument();
    expect(calls.some((c) => c.path.endsWith("/GetChanges") && (c.body as { subject?: string }).subject === "CHANGE_SUBJECT_TOPIC" && (c.body as { id?: string }).id === "newer")).toBe(true);
    // A fixed period asks with since.
    await user.click(within(since).getByRole("button", { name: strings.now.period }));
    await user.click(await screen.findByRole("option", { name: strings.now.periodWeek }));
    await waitFor(() => expect(calls.some((c) => c.path.endsWith("/GetChanges") && "since" in (c.body as object))).toBe(true));
    expect(calls.some((c) => c.path.endsWith("/MarkSeen"))).toBe(false);
    await user.click(within(since).getByRole("button", { name: strings.now.caughtUp }));
    const dialog = await screen.findByRole("alertdialog");
    expect(calls.some((c) => c.path.endsWith("/MarkSeen"))).toBe(false);
    await user.click(within(dialog).getByRole("button", { name: strings.now.caughtUp }));
    await waitFor(() => expect(calls.filter((c) => c.path.endsWith("/MarkSeen"))).toEqual([{ path: `${OPERATOR}/MarkSeen`, body: { all: true } }]));
    expect(await screen.findByText(strings.now.caughtUpDone)).toBeInTheDocument();
  });

  it("warns once, in one line, when items are open and nothing is delivered, linking to the switch (rulings V3)", async () => {
    wire({ [`${OPERATOR}/ListNeedsYou`]: needs({ writer: true, globalOn: false, hubRead: false }) });
    render("/");
    expect(await screen.findAllByText(strings.delivery.allOff)).toHaveLength(1);
    expect(screen.getByRole("link", { name: strings.delivery.switch })).toHaveAttribute("href", "/needs-you#delivery");
  });
});

describe("Desk", () => {
  it("shows which optional scenarios are down and what reads unknown, with no command to type, and how sessions start in plain words", async () => {
    wire({
      "/api/v1/capabilities/describe": { states: [{ id: "nh", name: "Notification Hub", status: "unavailable", message: "no answer", operatorCommand: WIRE.startCommand, uses: [{ id: "d", name: WIRE.mode }] }] },
      [`${ACTIVITY}/GetOverview`]: {
        researchHome: HOME,
        lenses: [
          { name: "research home", available: true, reason: HOME },
          { name: "web-console access", available: true, reason: WIRE.accessLens },
        ],
      },
      [`${OPERATOR}/ListNeedsYou`]: { items: [], courier: { writer: false, writerReason: WIRE.writerReason } },
      [`${LAUNCH}/GetLaunchOptions`]: { automaticAvailable: false, automaticReason: WIRE.automaticReason, webConsoleAccess: WIRE.access },
    });
    render("/desk");
    expect(await screen.findByText(strings.desk.unknownWhileDown)).toBeInTheDocument();
    expect(screen.queryByText(WIRE.startCommand)).not.toBeInTheDocument();
    expect(await screen.findByText(WIRE.writerReason)).toBeInTheDocument();
    expect(await screen.findByText(WIRE.automaticReason)).toBeInTheDocument();
    // Web Console access in plain words, never the raw lens row (V8).
    expect(screen.getByText(strings.access.availability)).toBeInTheDocument();
    expect(screen.queryByText(WIRE.accessLens)).not.toBeInTheDocument();
    expect(screen.getByText(strings.desk.neverRun, { exact: false })).toBeInTheDocument();
  });

  it("reads each scenario's health in plain words, and says when the probe itself fails", async () => {
    wire({
      "/api/v1/capabilities/describe": { states: [{ id: "am", name: "Agent Manager", status: "healthy", uses: [{ id: "s", name: "sessions" }] }, { id: WIRE.bareCapability, status: "degraded" }, { id: "x", name: "Other", status: "starting" }] },
      [`${LAUNCH}/GetLaunchTest`]: { at: AT, failedStep: "goal_sent", failedReason: "composer busy" },
    });
    render("/desk");
    expect(await screen.findByText(strings.desk.usedFor)).toBeInTheDocument();
    expect(screen.getByText(WIRE.bareCapability)).toBeInTheDocument();
    expect(await screen.findByText(strings.launch.failedAt, { exact: false })).toBeInTheDocument();
    cleanup();
    stubWire((path) => (path === "/api/v1/capabilities/describe" ? "{}" : path === `${LAUNCH}/GetLaunchTest` ? { running: true, at: AT } : {}));
    render("/desk");
    expect(await screen.findByText(strings.desk.noCapabilities)).toBeInTheDocument();
    expect(await screen.findByText(strings.desk.testRunning, { exact: false })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.desk.runLaunchTest })).toBeDisabled();
  });

  it("runs each self-test, launch and courier, only after confirming, and shows the courier off rather than green (rulings F15, V3)", async () => {
    const user = userEvent.setup();
    const calls = wire({
      [`${ACTIVITY}/GetOverview`]: { researchHome: HOME, lenses: [{ name: "courier", available: true, reason: WIRE.courierHealth }] },
      [`${OPERATOR}/ListNeedsYou`]: { items: [], courier: { writer: true, globalOn: false } },
      [`${LAUNCH}/GetLaunchTest`]: { at: AT, statusFinal: "complete", reportPath: `${HOME}/.desk/launch-test.json` },
      [`${OPERATOR}/TestCourier`]: { askId: "a1", key: "research-desk-test/x", firstDeliveredAt: AT, waitedSeconds: 4, expiresAt: AT },
    });
    const sent = () => calls.filter((c) => /\/(TestLaunch|TestCourier)$/.test(c.path)).map((c) => c.path);
    render("/desk");
    expect(await screen.findByText(strings.desk.testPassed, { exact: false })).toBeInTheDocument();
    expect(await screen.findByText(WIRE.courierHealth)).toBeInTheDocument();
    expect(await screen.findByText(strings.courier.off)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: strings.desk.runLaunchTest }));
    let dialog = await screen.findByRole("alertdialog");
    expect(sent()).toEqual([]);
    await user.click(within(dialog).getByRole("button", { name: strings.desk.runLaunchTest }));
    await waitFor(() => expect(sent()).toEqual([`${LAUNCH}/TestLaunch`]));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: strings.desk.runCourierTest }));
    dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(strings.desk.courierTestText)).toBeInTheDocument();
    expect(sent()).toEqual([`${LAUNCH}/TestLaunch`]);
    await user.click(within(dialog).getByRole("button", { name: strings.desk.runCourierTest }));
    await waitFor(() => expect(sent()).toEqual([`${LAUNCH}/TestLaunch`, `${OPERATOR}/TestCourier`]));
    expect(await screen.findByText(strings.desk.courierDelivered)).toBeInTheDocument();
  });

  it("says when no device received the test decision, and until when it stays open", async () => {
    const user = userEvent.setup();
    wire({ [`${OPERATOR}/TestCourier`]: { askId: "a1", waitedSeconds: 60, expiresAt: AT } });
    render("/desk");
    await user.click(await screen.findByRole("button", { name: strings.desk.runCourierTest }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: strings.desk.runCourierTest }));
    expect(await screen.findByText(strings.desk.courierNotDelivered)).toBeInTheDocument();
    expect(screen.getByText(strings.desk.courierExpires)).toBeInTheDocument();
  });
});

describe("One review", () => {
  const review = {
    id: "r1",
    folder: `${HOME}/reviews/r1`,
    title: { value: "Review one" },
    review: { url: "/api/v1/home/reviews/r1/REVIEW.md", file: { path: `${HOME}/reviews/r1/REVIEW.md` } },
    reviewHtml: { url: "/api/v1/home/reviews/r1/REVIEW.html" },
    proposals: [{ file: { path: `${HOME}/reviews/r1/proposals/t-MISSION.md` } }],
    topics: ["t"],
    verdicts: [{ at: AT, subject: "t", body: WIRE.verdict }],
  };

  it("shows the review text, its HTML a tap away, proposals, and the verdicts already recorded", async () => {
    wire({ [`${OPERATOR}/ListReviews`]: { reviews: [review] }, "/api/v1/home/reviews/r1/REVIEW.md": "## Accomplished\nSee [proposal](proposals/t-MISSION.md)." });
    render("/reviews/r1");
    expect(await screen.findByText(WIRE.verdict)).toBeInTheDocument();
    expect((await screen.findAllByRole("link", { name: /REVIEW\.html/ })).some((l) => l.getAttribute("target") === "_blank")).toBe(true);
    expect(await screen.findByRole("link", { name: "proposal" })).toHaveAttribute("href", "/files/reviews/r1/proposals/t-MISSION.md");
    expect(screen.getByText(strings.reviews.proposals)).toBeInTheDocument();
  });

  it("names a review that is not in the home", async () => {
    wire({ [`${OPERATOR}/ListReviews`]: { reviews: [] } });
    render("/reviews/gone");
    expect(await screen.findByText(strings.reviews.missing)).toBeInTheDocument();
  });
});
