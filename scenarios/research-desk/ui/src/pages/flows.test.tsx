/**
 * Outward flows driven through the real screens up to and through their
 * confirmation, with the API stubbed at the wire: what the operator is shown
 * before confirming, and exactly what is sent after.
 */
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders, stubWire, WireError, type WireCall } from "../test-utils";

const OPERATOR = "/vrooli.research_desk.v1.operator.OperatorService";
const ACTIVITY = "/vrooli.research_desk.v1.activity.ActivityService";
const LAUNCH = "/vrooli.research_desk.v1.launch.LaunchService";

/** Values the API sends; each must reach the screen verbatim. */
const WIRE = {
  recommendedProblem: "no Recommended: line",
  courierWhy: "courier on for c1",
  seenReason: "STATE.md not found",
  orphan: "AV9",
  legacyAsk: "pick a model",
  entrySubject: "AV4",
  kernelGoal: "Run the charter.",
  answerWarning: "STATE.md is busy",
  entryWarning: "steering is off",
  kernel: "Kernel text",
  gateError: "stance is 400 characters over",
  blocker: "old-1 is still live",
  uncertainReason: "STATE.md changed after the build",
  session: "s-42",
  relaunchId: "p1-r2",
  refusal: "a turn is running in b7117dc2; type /goal pause there yourself, or wind down through OPERATOR.md",
  reused: "ID reused, not delivered: AV4 was answered on 2026-11-10",
  courierHealth: "last pass 01:55Z; last delivery never; read-back 01:55Z",
  unreadable: "cannot read STATE.md: permission denied",
  recommendedLabel: "Keep it uninstalled",
  quietBlocker: "a: q-1 state unknown",
  threeTopics: ["Topic a", "Topic b", "Topic c"],
  first: "First",
  twin: "Same ID elsewhere",
  globalTitle: "Global one",
  unknownBlocker: "p1 is unknown, not prepared; a relaunch is a new commission (`research-desk build --relaunch-of p1`)",
  unknownBlockerWords: "p1 is unknown, not prepared; a relaunch is a new commission",
  dryText: "### 2026-11-12T08:00:00Z · exact text from the API's dry run",
  readNext: "It is read at the start of the next unit.",
  readAfterResume: "Paused: it is read at the next unit after the supervisor is resumed.",
  instruction: "Approve the grant in Notification Hub.",
  linkLabel: "the grant decision",
  liveReason: "it is live: the operator chooses",
  supersedeReason: "paused and its allowance ended at 2026-11-11T08:00Z",
  suggestedQuote: "make it more reliable than it is now",
  quoteSource: "/home/u/.vrooli/research/cohorts/old-1/STANCE-MESSAGE.txt",
  workstreamTitle: "Measure local models before tuning",
  neverReason: "no session or goal for p1 in 2 Codex homes",
  neverRefused: "p1 has a goal in thread t-9",
} as const;

type Resolutions = { commissionId: string; choice: string }[];
const chosenIn = (body: unknown, id: string) => ((body as { resolutions?: Resolutions } | undefined)?.resolutions ?? []).find((r) => r.commissionId === id)?.choice;

/** An earlier commission's verdict as the API judges it with the request's choice (rulings P2). */
function verdict(topic: string, commissionId: string, state: string, body: unknown, recommended = "", recommendedReason = "") {
  const chosen = chosenIn(body, commissionId);
  return {
    topic,
    commissionId,
    state,
    blocks: !chosen,
    message: `${state}${chosen ? "" : "; choose supersede or run alongside"}`,
    choices: ["RESOLUTION_SUPERSEDE", "RESOLUTION_RUN_ALONGSIDE"],
    recommended: recommended || undefined,
    recommendedReason,
    chosen,
  };
}

/** The API's dry run: the exact text it would append and when it is read (rulings P5). */
const dryRun = (body: unknown) => ((body as { dryRun?: boolean } | undefined)?.dryRun ? { text: WIRE.dryText, written: false, reading: { message: WIRE.readNext } } : undefined);

/** Requests that change something; a dry run writes nothing. */
const writes = (calls: WireCall[]) =>
  calls.filter((c) => /\/(Answer|AddEntry|Control|SetCourier|Build|Launch|EnableSteering|AddVerdict|MarkSeen)$/.test(c.path) && !(c.body as { dryRun?: boolean } | undefined)?.dryRun);

const render = (path: string) => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

afterEach(() => {
  cleanup();
  // The launch form waits in session storage between visits (rulings V1); each test starts empty.
  sessionStorage.clear();
});

describe("Needs you", () => {
  const item = {
    id: "AV4",
    title: "Choose how AV4 handles yt-dlp",
    body: "Keep it uninstalled or ask for a proposal.",
    commissionId: "programme-five-20261112",
    options: [
      { key: "keep", label: "Keep it uninstalled" },
      { key: "propose", label: "Ask for a proposal" },
    ],
    recommended: { key: "keep", label: "Keep it uninstalled" },
    answerable: true,
  };

  it("shows the exact entry before answering, and sends only after confirming", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path, body) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [item], courier: { writer: true } } : (dryRun(body) ?? {})));
    render("/needs-you");
    await user.click(await screen.findByRole("button", { name: "Keep it uninstalled" }));
    const dialog = await screen.findByRole("alertdialog");
    // The exact text is the API's, never rebuilt here, with when it will be read.
    expect(await within(dialog).findByText(WIRE.dryText)).toBeInTheDocument();
    expect(within(dialog).getByText(WIRE.readNext)).toBeInTheDocument();
    expect(calls).toContainEqual({ path: `${OPERATOR}/Answer`, body: { commissionId: item.commissionId, itemId: "AV4", optionKey: "keep", dryRun: true } });
    expect(writes(calls)).toHaveLength(0);
    await user.type(within(dialog).getByRole("textbox"), "tell me if AV6 needs it");
    await waitFor(() => expect(calls).toContainEqual({ path: `${OPERATOR}/Answer`, body: { commissionId: item.commissionId, itemId: "AV4", optionKey: "keep", note: "tell me if AV6 needs it", dryRun: true } }));
    await user.click(within(dialog).getByRole("button", { name: strings.needsYou.confirm }));
    await waitFor(() =>
      expect(writes(calls)).toEqual([
        { path: `${OPERATOR}/Answer`, body: { commissionId: item.commissionId, itemId: "AV4", optionKey: "keep", note: "tell me if AV6 needs it" } },
      ]),
    );
  });

  it("never offers a text box for a secret, and says where it goes", async () => {
    stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [{ ...item, secret: "vault: research/hub-token" }], courier: { writer: true } } : {}));
    render("/needs-you");
    expect(await screen.findByText(strings.needsYou.secretTitle)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Keep it uninstalled" })).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("says plainly when the desk is not authorized", async () => {
    // A Connect error is a non-200 answer with a code; the API refuses this session.
    vi.stubGlobal("fetch", () => Promise.resolve(new Response(JSON.stringify({ code: "permission_denied", message: "no" }), { status: 403, headers: { "Content-Type": "application/json" } })));
    render("/needs-you");
    expect(await screen.findByText(strings.delivery.unauthorizedTitle)).toBeInTheDocument();
  });
});

describe("Needs you feed", () => {
  const base = { commissionId: "c1", options: [], sources: [] };
  const feed = {
    items: [
      { ...base, id: "Q1", title: "Delivered ask", delivered: { body: "sent" }, recommendedProblem: WIRE.recommendedProblem, packet: "/home/u/.vrooli/research/cohorts/c1/Q1.md", sources: ["/home/u/.vrooli/research/cohorts/c1/STATE.md"] },
      { ...base, id: "Q2", title: "Queued ask", courier: { on: true, why: WIRE.courierWhy }, deliveryReason: WIRE.courierWhy },
      { ...base, id: "Q3", title: "Local only", delivery: "DELIVERY_NONE", deliveredByScheduler: "sent at 09:30" },
      { ...base, commissionId: "", id: "Q4", title: "Not sent", notAnswerableReason: "no commission folder", instruction: WIRE.instruction, links: [{ label: WIRE.linkLabel, target: "/home/u/.vrooli/research/ATTENTION.md" }] },
      { ...base, id: "A1", title: "Seen answer", answer: { at: "2026-11-11T09:30:00Z", body: "keep" }, seen: { value: true } },
      { ...base, id: "A2", title: "Unplaceable answer", answer: { body: "drop" }, seen: { evidence: { known: false, reason: WIRE.seenReason } } },
      { ...base, id: "A3", title: "Unseen answer", answer: { body: "wait" } },
    ],
    legacy: [{ source: "/home/u/.vrooli/research/cohorts/old/STATE.md", text: `**Needs you:** ${WIRE.legacyAsk}` }],
    courier: { writer: true, globalOn: true, hubRead: true, orphans: [{ key: WIRE.orphan, askState: "open", why: "item gone" }], switchesFile: "/home/u/.vrooli/research/.desk/courier.json" },
  };

  it("shows each item's delivery, answer and whether the supervisor has seen it", async () => {
    stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? feed : {}));
    render("/needs-you");
    expect(await screen.findByText(strings.needsYou.badgeDelivered)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.badgeQueued)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.badgeHere)).toBeInTheDocument();
    expect(screen.getAllByText(strings.needsYou.badgeNotSent).length).toBeGreaterThan(0);
    expect(screen.getByText(WIRE.recommendedProblem)).toBeInTheDocument();
    // Items the desk cannot answer say where they are answered, in the item's own words and links (rulings V2, P6).
    expect(screen.getAllByText(strings.needsYou.answeredWhere)).toHaveLength(4);
    expect(screen.getByText(WIRE.instruction)).toBeInTheDocument();
    expect(screen.getByText(WIRE.linkLabel)).toBeInTheDocument();
    // The same reason from two places is said once.
    expect(screen.getByText(WIRE.courierWhy)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.deliveredByScheduler)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.seen)).toBeInTheDocument();
    expect(screen.getByText(WIRE.seenReason)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.notSeen)).toBeInTheDocument();
    expect(screen.getByText(WIRE.legacyAsk, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(WIRE.orphan)).toBeInTheDocument();
  });

  it("shows each item as one collapsed row, and opens the one a link names by commission plus ID (rulings V2)", async () => {
    const twins = {
      items: [
        { ...base, id: "Q1", title: WIRE.first, recommended: { key: "keep", label: WIRE.recommendedLabel } },
        { ...base, commissionId: "c2", id: "Q1", title: WIRE.twin },
        { ...base, commissionId: "", id: "G1", title: WIRE.globalTitle },
      ],
      courier: feed.courier,
    };
    stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? twins : {}));
    const scrolled = vi.fn();
    Object.defineProperty(Element.prototype, "scrollIntoView", { configurable: true, value: scrolled });
    render("/needs-you#c2%2FQ1");
    const target = await screen.findByText(WIRE.twin);
    const row = (title: HTMLElement) => title.closest("li") as HTMLElement;
    const first = row(screen.getByText(WIRE.first));
    expect(row(target).id).toBe("c2/Q1");
    expect(first.id).toBe("c1/Q1");
    expect(row(screen.getByText(WIRE.globalTitle)).id).toBe("global/G1");
    // Collapsed, with the recommended answer on the row; only the linked one is open, and it is scrolled to.
    expect(first.querySelector("details")?.open).toBe(false);
    expect(within(first).getByText(WIRE.recommendedLabel).closest("summary")).not.toBeNull();
    await waitFor(() => expect(row(target).querySelector("details")?.open).toBe(true));
    expect(scrolled).toHaveBeenCalled();
    Reflect.deleteProperty(Element.prototype, "scrollIntoView");
  });

  it("shows a reused ID as not delivered, with what to do instead of an answer form", async () => {
    const reused = { ...base, id: "AV4", title: "Reused ask", answerable: true, options: [{ key: "a", label: "A" }], deliveryReason: WIRE.reused, deliveryReasonCode: "DELIVERY_REASON_ID_REUSED" };
    stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [reused], courier: { writer: true, globalOn: true, hubRead: true } } : {}));
    render("/needs-you");
    expect(await screen.findByText(strings.needsYou.badgeReused)).toBeInTheDocument();
    expect(screen.getByText(strings.needsYou.reusedTitle)).toBeInTheDocument();
    expect(screen.getByText(WIRE.reused)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: strings.needsYou.reusedAction })).toHaveAttribute("href", "/commissions/c1?tab=steer");
    expect(screen.queryByRole("button", { name: strings.needsYou.ownWords })).not.toBeInTheDocument();
  });

  it("says which files it cannot read instead of an empty feed, and shows the courier's real state", async () => {
    stubWire((path) =>
      path === `${OPERATOR}/ListNeedsYou`
        ? { items: [], unreadable: [{ source: "/home/u/.vrooli/research/cohorts/c1/STATE.md", reason: WIRE.unreadable }], courier: { writer: true, globalOn: true, hubRead: true } }
        : path === `${ACTIVITY}/GetOverview`
          ? { lenses: [{ name: "courier", available: true, reason: WIRE.courierHealth }] }
          : {},
    );
    render("/needs-you");
    expect(await screen.findByText(WIRE.unreadable)).toBeInTheDocument();
    expect(screen.queryByText(strings.needsYou.emptyTitle)).not.toBeInTheDocument();
    expect(await screen.findByText(WIRE.courierHealth)).toBeInTheDocument();
    expect(screen.getByText(strings.courier.healthy)).toBeInTheDocument();
  });

  it("says so when nothing is waiting", async () => {
    stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [], courier: { writer: true, globalOn: true, hubRead: true } } : {}));
    render("/needs-you");
    expect(await screen.findByText(strings.needsYou.emptyTitle)).toBeInTheDocument();
  });

  it("needs words before an own-words answer, and sends them as 'other'", async () => {
    const user = userEvent.setup();
    const ask = { ...base, id: "AV5", title: "Pick", answerable: true, options: [{ key: "a", label: "A" }], courier: { askId: "ask-1" } };
    const calls = stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [ask], courier: { writer: true, globalOn: true, hubRead: true } } : path === `${OPERATOR}/Answer` ? { warning: WIRE.answerWarning } : {}));
    render("/needs-you");
    await user.click(await screen.findByRole("button", { name: strings.needsYou.ownWords }));
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(strings.needsYou.askStaysOpen)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: strings.needsYou.confirm }));
    expect(await within(dialog).findByText(strings.needsYou.ownWordsRequired)).toBeInTheDocument();
    expect(writes(calls)).toHaveLength(0);
    await user.type(within(dialog).getByRole("textbox"), "Neither; wait a day.");
    await user.click(within(dialog).getByRole("button", { name: strings.needsYou.confirm }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/Answer`, body: { commissionId: "c1", itemId: "AV5", optionKey: "other", note: "Neither; wait a day." } }]));
    expect(await screen.findByText(WIRE.answerWarning)).toBeInTheDocument();
  });

  it("turns global delivery off only after confirming", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${OPERATOR}/ListNeedsYou` ? { items: [], courier: { writer: true, globalOn: true, hubRead: true } } : {}));
    render("/needs-you");
    await user.click(await screen.findByRole("switch", { name: strings.courier.globalLabel }));
    const dialog = await screen.findByRole("alertdialog");
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.courier.turnOffLabel }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/SetCourier`, body: { global: true } }]));
  });
});

describe("Commission controls", () => {
  const commission = (state: string, windowOpen: boolean) => ({
    commission: { id: "c1", folder: "/home/u/.vrooli/research/cohorts/c1", state, title: { value: "C one" } },
    matches: [{ kind: "MATCH_KIND_EXACT", webConsoleSessionId: "b7117dc2", windowOpen, basis: "launch-record" }],
    steering: { value: true, evidence: { known: true } },
  });

  it("names the keystrokes and session before pausing", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_LIVE", true) : {}));
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.control.labels.pause }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog.querySelector("pre")?.textContent).toBe("/goal pause ⏎");
    await user.click(within(dialog).getByRole("button", { name: strings.control.labels.pause }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Control`, body: { commissionId: "c1", action: "CONTROL_ACTION_PAUSE" } }]));
  });

  it("shows a refused pause with its reason and the keystrokes to type by hand, and offers wind down instead", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) =>
      path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_LIVE", true) : path === `${LAUNCH}/Control` ? new WireError("failed_precondition", WIRE.refusal) : {},
    );
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.control.labels.pause }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: strings.control.labels.pause }));
    expect(await screen.findByText(WIRE.refusal)).toBeInTheDocument();
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(screen.getByText(strings.control.refusedTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.control.byHand).nextElementSibling?.textContent).toBe("/goal pause ⏎");
    await user.click(screen.getByRole("button", { name: strings.control.windDownInstead }));
    const field = screen.getByLabelText(strings.say.windDownLabel);
    expect(field).toHaveFocus();
    expect(writes(calls).map((c) => c.path)).toEqual([`${LAUNCH}/Control`]);
  });

  it("keeps any other control failure in its dialog", async () => {
    const user = userEvent.setup();
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_LIVE", true) : path === `${LAUNCH}/Control` ? new WireError("internal", WIRE.refusal, 500) : {}));
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.control.labels.pause }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: strings.control.labels.pause }));
    expect(await within(dialog).findByText(WIRE.refusal)).toBeInTheDocument();
    expect(screen.queryByText(strings.control.refusedTitle)).not.toBeInTheDocument();
  });

  it("counts an entry against the 4,000-character limit and holds one that is over it", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_LIVE", true) : {}));
    render("/commissions/c1?tab=steer");
    const field = await screen.findByLabelText(strings.say.directionLabel);
    expect(screen.getByText(strings.entry.count)).toBeInTheDocument();
    await user.click(field);
    await user.paste("x".repeat(4001));
    expect(await screen.findByText(strings.entry.tooLong)).toBeInTheDocument();
    expect(field).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("button", { name: strings.say.review })).toBeDisabled();
    await user.type(field, "{Backspace}");
    expect(screen.queryByText(strings.entry.tooLong)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.say.review })).toBeEnabled();
    expect(writes(calls)).toHaveLength(0);
  });

  it("offers no keystroke control without an open exactly-matched window", async () => {
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_PAUSED", false) : {}));
    render("/commissions/c1?tab=steer");
    expect(await screen.findByRole("button", { name: strings.control.labels.resume })).toBeDisabled();
    expect(screen.getByRole("button", { name: strings.control.labels.stop })).toBeDisabled();
    expect(screen.getByRole("button", { name: strings.control.labels.discard })).toBeEnabled();
  });

  it("refuses to discard without a reason", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_PREPARED", false) : {}));
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.control.labels.discard }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: strings.control.labels.discard }));
    expect(await within(dialog).findByText(strings.confirm.reasonRequired)).toBeInTheDocument();
    expect(writes(calls)).toHaveLength(0);
    await user.type(within(dialog).getByRole("textbox"), "wrong topics");
    await user.click(within(dialog).getByRole("button", { name: strings.control.labels.discard }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Control`, body: { commissionId: "c1", action: "CONTROL_ACTION_DISCARD", reason: "wrong topics" } }]));
  });

  it("writes direction to OPERATOR.md only after showing the entry", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path, body) => (path === `${ACTIVITY}/GetCommission` ? commission("COMMISSION_STATE_LIVE", true) : (dryRun(body) ?? {})));
    render("/commissions/c1?tab=steer");
    await user.type(await screen.findByLabelText(strings.say.directionLabel), "Focus on IT6.");
    await user.click(screen.getByRole("button", { name: strings.say.review }));
    const dialog = await screen.findByRole("alertdialog");
    expect(await within(dialog).findByText(WIRE.dryText)).toBeInTheDocument();
    expect(calls).toContainEqual({ path: `${OPERATOR}/AddEntry`, body: { commissionId: "c1", kind: "ENTRY_KIND_DIRECTION", text: "Focus on IT6.", dryRun: true } });
    await user.click(within(dialog).getByRole("button", { name: strings.say.confirm }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/AddEntry`, body: { commissionId: "c1", kind: "ENTRY_KIND_DIRECTION", text: "Focus on IT6." } }]));
  });
});

describe("Commission steering", () => {
  const folder = "/home/u/.vrooli/research/cohorts/c1";
  const detail = (extra: Record<string, unknown> = {}) => ({
    commission: { id: "c1", folder, state: "COMMISSION_STATE_LIVE", title: { value: "C one" }, entryReading: { when: "READ_WHEN_NEXT_UNIT", message: WIRE.readNext, readSince: { value: "2026-11-11T10:00:00Z" } } },
    matches: [{ kind: "MATCH_KIND_EXACT", webConsoleSessionId: "b7117dc2", windowOpen: true }],
    files: [{ name: "STATE.md", path: `${folder}/STATE.md`, modified: "2026-11-11T10:00:00Z" }],
    steering: { value: false, evidence: { known: true } },
    operatorEntries: [
      { at: "2026-11-11T09:00:00Z", kind: "direction", via: "desk", body: "Read first" },
      { at: "2026-11-11T11:00:00Z", kind: "answer", subject: WIRE.entrySubject, via: "hub", body: "keep" },
      { kind: "direction", via: "hand", body: "undated" },
    ],
    operatorUnparsed: 1,
    ...extra,
  });

  it("marks each entry read once the API's read-since passed it, and unknown without a time", async () => {
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? detail() : {}));
    render("/commissions/c1?tab=steer");
    expect(await screen.findByText(strings.say.seen)).toBeInTheDocument();
    expect(screen.getByText(WIRE.readNext)).toBeInTheDocument();
    expect(screen.getByText(strings.say.notSeen)).toBeInTheDocument();
    expect(screen.getByText(WIRE.entrySubject, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(strings.say.unparsed)).toBeInTheDocument();
  });

  it("says a paused supervisor reads a new entry only after it is resumed (rulings V4)", async () => {
    const paused = { id: "c1", folder, state: "COMMISSION_STATE_PAUSED", entryReading: { when: "READ_WHEN_AFTER_RESUME", message: WIRE.readAfterResume } };
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? detail({ commission: paused, steering: { value: true } }) : {}));
    render("/commissions/c1?tab=steer");
    const line = await screen.findByText(WIRE.readAfterResume);
    expect(line.closest("p")).toHaveClass("rd-reading-waits");
    expect(screen.queryByText(WIRE.readNext)).not.toBeInTheDocument();
  });

  it("adds the steering sentence to an older charter only after showing it", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? detail() : {}));
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.say.enableSteering }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog.querySelector("pre")?.textContent).toMatch(/^## Steering\n\nRead entries of `OPERATOR.md`/);
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.say.enableSteering }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/EnableSteering`, body: { commissionId: "c1" } }]));
  });

  it("writes a wind-down entry and shows the API's warning", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path, body) =>
      path === `${ACTIVITY}/GetCommission` ? detail({ steering: { evidence: { known: false, reason: "no README" } } }) : (dryRun(body) ?? (path === `${OPERATOR}/AddEntry` ? { warning: WIRE.entryWarning } : {})),
    );
    render("/commissions/c1?tab=steer");
    expect(await screen.findByText(strings.say.steeringUnknown)).toBeInTheDocument();
    await user.click(screen.getByRole("radio", { name: strings.say.windDown }));
    await user.type(screen.getByLabelText(strings.say.windDownLabel), "Land IT6, then stop.");
    await user.click(screen.getByRole("button", { name: strings.say.review }));
    const dialog = await screen.findByRole("alertdialog");
    expect(await within(dialog).findByText(WIRE.dryText)).toBeInTheDocument();
    expect(calls).toContainEqual({ path: `${OPERATOR}/AddEntry`, body: { commissionId: "c1", kind: "ENTRY_KIND_WIND_DOWN", text: "Land IT6, then stop.", dryRun: true } });
    await user.click(within(dialog).getByRole("button", { name: strings.say.confirm }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/AddEntry`, body: { commissionId: "c1", kind: "ENTRY_KIND_WIND_DOWN", text: "Land IT6, then stop." } }]));
    expect(await screen.findByText(WIRE.entryWarning)).toBeInTheDocument();
  });

  it("re-sends the stance and reports what the session said", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? detail({ steering: { value: true } }) : path === `${LAUNCH}/Control` ? { message: "stance re-sent", goalStatus: "active" } : {}));
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("button", { name: strings.control.labels.resend }));
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(strings.control.keptSlot)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: strings.control.labels.resend }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Control`, body: { commissionId: "c1", action: "CONTROL_ACTION_RESEND_STANCE" } }]));
    expect(await screen.findByText(/stance re-sent/)).toBeInTheDocument();
  });

  it("turns this commission's delivery on only after confirming", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) =>
      path === `${ACTIVITY}/GetCommission`
        ? detail({ steering: { value: true } })
        : path === `${OPERATOR}/ListNeedsYou`
          ? { items: [], courier: { writer: true, switchesFile: "/home/u/.vrooli/research/.desk/courier.json", commissions: [{ commissionId: "c1", on: false, reason: "off by default" }] } }
          : {},
    );
    render("/commissions/c1?tab=steer");
    await user.click(await screen.findByRole("switch", { name: new RegExp(strings.courier.switchLabel) }));
    const dialog = await screen.findByRole("alertdialog");
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.courier.turnOnLabel }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${OPERATOR}/SetCourier`, body: { commissionId: "c1", on: true } }]));
  });

  it("offers no new words once a commission has ended", async () => {
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? detail({ commission: { id: "c1", folder, state: "COMMISSION_STATE_ENDED" }, operatorEntries: [], operatorUnparsed: 0 }) : {}));
    render("/commissions/c1?tab=steer");
    expect(await screen.findByText(strings.say.ended)).toBeInTheDocument();
    expect(screen.getByText(strings.say.noEntries)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.control.labels.discard })).toBeDisabled();
  });
});

describe("Review verdicts", () => {
  it("records a verdict on a topic the review has no section for, only after confirming the exact entry", async () => {
    const user = userEvent.setup();
    const review = { id: "r1", folder: "/home/u/.vrooli/research/reviews/r1", title: { value: "Review one" }, topics: [], verdicts: [], proposals: [] };
    const calls = stubWire((path, body) =>
      path === `${OPERATOR}/ListReviews`
        ? { reviews: [review] }
        : path === `${ACTIVITY}/ListTopics`
          ? { topics: [{ topic: "local-model-performance", title: { value: "Local models" } }] }
          : (dryRun(body) ?? {}),
    );
    render("/reviews/r1");
    await user.click(await screen.findByRole("button", { name: new RegExp(strings.reviews.otherTopic) }));
    await user.click(await screen.findByRole("option", { name: "Local models" }));
    await user.type(await screen.findByLabelText(strings.reviews.verdictLabel), "Keep going; measure first.");
    await user.click(screen.getByRole("button", { name: strings.reviews.record }));
    const dialog = await screen.findByRole("alertdialog");
    expect(await within(dialog).findByText(WIRE.dryText)).toBeInTheDocument();
    expect(calls).toContainEqual({ path: `${OPERATOR}/AddVerdict`, body: { reviewId: "r1", topic: "local-model-performance", text: "Keep going; measure first.", dryRun: true } });
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.reviews.record }));
    await waitFor(() =>
      expect(writes(calls)).toEqual([{ path: `${OPERATOR}/AddVerdict`, body: { reviewId: "r1", topic: "local-model-performance", text: "Keep going; measure first." } }]),
    );
  });
});

describe("Launch", () => {
  const topics = {
    topics: [
      {
        topic: "local-model-performance",
        title: { value: "Local models" },
        commissionIds: ["old-1"],
        liveCommissionIds: ["old-1"],
        pausedCommissionIds: [],
        limitedCommissionIds: [],
        missionCheck: { mode: "research", workstreams: [{ id: "IT6", status: "active" }, { id: "IT7", status: "landed" }] },
      },
    ],
  };
  const options = { automaticAvailable: true, defaultSupervisorModel: "gpt-5", defaultSupervisorEffort: "high", defaultOwnerModelTier: "inherit", supervisorModels: ["gpt-5"], supervisorEfforts: ["high"], ownerModelTiers: ["inherit"] };
  const preflight = (ok: boolean, body?: unknown) => {
    const old = verdict("local-model-performance", "old-1", "live", body, "", WIRE.liveReason);
    return {
      packet: {
        ok,
        commissionId: "local-models-w1-20261112",
        gates: [
          { name: "charter_bytes", value: "9000", limit: "24000", ok: true },
          { name: "stance_chars", value: ok ? "3100" : "3900", limit: "3500", ok },
        ],
        cards: [{ topic: "local-model-performance", order: ["IT6"] }],
        texts: { stance: "Stance text. Reply \"ready\" and wait", kernel: WIRE.kernel, charter: "# Charter" },
        errors: ok ? [] : [{ code: "gate_overflow", message: WIRE.gateError }],
      },
      topics: [old],
      launchable: ok && !old.blocks,
      blockers: [...(ok ? [] : [WIRE.blocker]), ...(old.blocks ? [`local-model-performance: old-1 ${old.message}`] : [])],
    };
  };

  async function fill(user: ReturnType<typeof userEvent.setup>) {
    await user.click(await screen.findByRole("checkbox", { name: /Local models/ }));
    await user.click(screen.getByRole("button", { name: strings.launch.addId }));
    await user.type(screen.getByLabelText(strings.launch.quoteN), "make it reliable");
  }

  it("previews the packet, then builds with the operator's choice and launches only after confirming", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path, body) =>
      path === `${LAUNCH}/GetLaunchOptions`
        ? options
        : path === `${ACTIVITY}/ListTopics`
          ? topics
          : path === `${LAUNCH}/Preflight`
            ? preflight(true, body)
            : path === `${LAUNCH}/Build`
              ? { packet: { ok: true, commissionId: "local-models-w1-20261112" } }
              : {},
    );
    render("/launch");
    await fill(user);
    // A live earlier commission has no recommendation: the launch waits until the operator chooses, on this screen.
    expect(await screen.findByText(strings.preflight.holdNote, undefined, { timeout: 3000 })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: strings.preflight.choiceHold })).toBeChecked();
    expect(screen.getByText(WIRE.liveReason)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.launch.launch })).toBeDisabled();
    await user.click(screen.getByRole("radio", { name: strings.preflight.choiceAlongside }));
    expect(await screen.findByText(WIRE.kernel)).toBeInTheDocument();
    const launch = screen.getByRole("button", { name: strings.launch.launch });
    await waitFor(() => expect(launch).toBeEnabled(), { timeout: 3000 });
    const asked = calls.filter((c) => c.path === `${LAUNCH}/Preflight`).at(-1)?.body as { specJson: string; resolutions: Resolutions };
    expect(asked.resolutions).toEqual([{ commissionId: "old-1", choice: "RESOLUTION_RUN_ALONGSIDE" }]);
    // The choice travels in the request; the spec holds only the operator's packet.
    const spec = JSON.parse(asked.specJson);
    expect(spec).toMatchObject({ topics: ["local-model-performance"], order: { "local-model-performance": ["IT6"] }, stance_quotes: ["make it reliable"], allowance_hours: 24 });
    expect(spec).not.toHaveProperty("run_alongside");
    expect(writes(calls)).toHaveLength(0);
    await user.click(launch);
    const dialog = await screen.findByRole("alertdialog");
    // The one confirmation names what happens to the earlier commission and how Web Console is reached.
    expect(within(dialog).getByText(strings.launch.willRunAlongside)).toBeInTheDocument();
    expect(within(dialog).getByText(strings.access.unknown)).toBeInTheDocument();
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.launch.launch }));
    await waitFor(() => expect(writes(calls).map((c) => c.path)).toEqual([`${LAUNCH}/Build`, `${LAUNCH}/Launch`]));
    expect(writes(calls)[0]?.body).toMatchObject({ resolutions: [{ commissionId: "old-1", choice: "RESOLUTION_RUN_ALONGSIDE" }] });
    expect(writes(calls)[1]?.body).toEqual({ commissionId: "local-models-w1-20261112" });
  });

  it("holds the launch while a gate fails, and says what to cut", async () => {
    const user = userEvent.setup();
    stubWire((path, body) => (path === `${LAUNCH}/GetLaunchOptions` ? options : path === `${ACTIVITY}/ListTopics` ? topics : path === `${LAUNCH}/Preflight` ? preflight(false, body) : {}));
    render("/launch");
    expect(await screen.findByText(strings.launch.needTopics)).toBeInTheDocument();
    await fill(user);
    expect(await screen.findByText(WIRE.gateError, undefined, { timeout: 3000 })).toBeInTheDocument();
    // The blocker is a choice on this screen, in plain words, not the API's line with its spec field.
    expect(screen.queryByText(WIRE.blocker)).not.toBeInTheDocument();
    expect(screen.getByText(strings.preflight.stateLive)).toBeInTheDocument();
    expect(screen.getByText(strings.preflight.gatesTitle).closest("details")).toHaveAttribute("open");
    expect(screen.getByRole("button", { name: strings.launch.launch })).toBeDisabled();
  });

  it("sends the operator's card order, notes, quotes, slug and model choice, and builds a manual packet only after confirming", async () => {
    const user = userEvent.setup();
    const two = { topics: [{ ...topics.topics[0], missionCheck: { mode: "research", workstreams: [{ id: "IT6", status: "active", title: WIRE.workstreamTitle }, { id: "IT8", status: "open" }] } }] };
    const calls = stubWire((path) =>
      path === `${LAUNCH}/GetLaunchOptions`
        ? { ...options, supervisorModels: ["gpt-5", "gpt-5-pro"] }
        : path === `${ACTIVITY}/ListTopics`
          ? two
          : path === `${LAUNCH}/Preflight`
            ? { ...preflight(true), launchable: true, blockers: [], topics: [{ topic: "local-model-performance", commissionId: "old-1", message: "ended", readFirst: "/home/u/.vrooli/research/cohorts/old-1/STATE.md" }] }
            : path === `${LAUNCH}/Build`
              ? { packet: { ok: true, commissionId: "local-models-w1-20261112" } }
              : {},
    );
    render("/launch");
    await user.click(await screen.findByRole("checkbox", { name: /Local models/ }));
    const add = within(screen.getByRole("list", { name: strings.launch.addWorkstream }));
    // Workstreams read by their titles, not bare IDs (rulings P3).
    expect(add.getByText(WIRE.workstreamTitle)).toBeInTheDocument();
    await user.click(add.getAllByRole("button")[0] as HTMLElement);
    await user.click(screen.getByRole("button", { name: strings.launch.addId }));
    // IT8 was added second; moving it up puts it first, then IT6 is removed.
    await user.click(screen.getAllByRole("button", { name: strings.launch.moveUp })[1] as HTMLElement);
    await user.click(screen.getAllByRole("button", { name: strings.launch.removeWorkstream })[1] as HTMLElement);
    await user.type(screen.getByLabelText(strings.launch.cardNote), "measure first");
    await user.type(screen.getByLabelText(strings.launch.emphasis), "reliability");
    await user.click(screen.getByRole("button", { name: strings.launch.addQuote }));
    const quotes = screen.getAllByLabelText(strings.launch.quoteN);
    await user.type(quotes[0] as HTMLElement, "first words");
    await user.type(quotes[1] as HTMLElement, "dropped");
    await user.click(screen.getAllByRole("button", { name: strings.launch.removeQuote })[1] as HTMLElement);
    await user.type(screen.getByLabelText(new RegExp(`^${strings.launch.slug}`)), "LOCAL-Models");
    await user.click(screen.getByRole("button", { name: strings.launch.supervisorModel }));
    await user.click(await screen.findByRole("option", { name: "gpt-5-pro" }));
    expect(screen.getByText(strings.launch.costModel)).toBeInTheDocument();
    // The newest checkpoint is offered already ticked: the desk's recommendation, changed here if wanted.
    expect(await screen.findByRole("checkbox", { name: strings.preflight.readFirstOn }, { timeout: 3000 })).toBeChecked();
    await waitFor(
      () => {
        const last = calls.filter((c) => c.path === `${LAUNCH}/Preflight`).at(-1)?.body as { specJson: string };
        expect(JSON.parse(last.specJson)).toEqual({
          topics: ["local-model-performance"],
          order: { "local-model-performance": ["IT8"] },
          card_notes: { "local-model-performance": "measure first" },
          emphasis: "reliability",
          stance_quotes: ["first words"],
          allowance_hours: 24,
          supervisor_model: "gpt-5-pro",
          supervisor_effort: "high",
          owner_model_tier: "inherit",
          read_first: { "local-model-performance": "/home/u/.vrooli/research/cohorts/old-1/STATE.md" },
          slug: "local-models",
        });
      },
      { timeout: 3000 },
    );
    const manual = screen.getByRole("button", { name: strings.launch.buildManual });
    await waitFor(() => expect(manual).toBeEnabled(), { timeout: 3000 });
    await user.click(manual);
    const dialog = await screen.findByRole("alertdialog");
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.launch.buildManual }));
    await waitFor(() => expect(writes(calls).map((c) => c.path)).toEqual([`${LAUNCH}/Build`]));
  });

  it("launches three topics with the desk's defaults in five taps: tick, tick, tick, Launch, Confirm (rulings V1)", async () => {
    const user = userEvent.setup();
    const three = { topics: ["a", "b", "c"].map((name) => ({ ...topics.topics[0], topic: name, title: { value: `Topic ${name}` }, liveCommissionIds: [] })) };
    const calls = stubWire((path, body) => {
      if (path === `${LAUNCH}/GetLaunchOptions`) {
        // Suggestions come for the topics asked about (rulings P3).
        const asked = (body as { topics?: string[] }).topics ?? [];
        return asked.length > 0 ? { ...options, stanceQuotes: [{ text: WIRE.suggestedQuote, source: WIRE.quoteSource, commissionId: "old-1" }] } : options;
      }
      if (path === `${ACTIVITY}/ListTopics`) return three;
      if (path === `${LAUNCH}/Preflight`) {
        const paused = verdict("a", "p-4", "paused", body, "RESOLUTION_SUPERSEDE", WIRE.supersedeReason);
        return { ...preflight(true), topics: [paused], launchable: !paused.blocks, blockers: paused.blocks ? [`a: p-4 ${paused.message}`] : [] };
      }
      if (path === `${LAUNCH}/Build`) return { packet: { ok: true, commissionId: "a-w1-20261112" } };
      return {};
    });
    render("/launch");
    let taps = 0;
    const tap = async (el: HTMLElement) => {
      taps += 1;
      await user.click(el);
    };
    for (const name of WIRE.threeTopics) await tap(await screen.findByRole("checkbox", { name }));
    // The quotes are filled from the newest commission on these topics, and say where from.
    expect(await screen.findByDisplayValue(WIRE.suggestedQuote)).toBeInTheDocument();
    expect(screen.getByText(strings.launch.quotesFrom)).toBeInTheDocument();
    // The paused commission past its allowance arrives with supersede selected, and why.
    expect(await screen.findByRole("radio", { name: new RegExp(strings.preflight.choiceSupersede) }, { timeout: 3000 })).toBeChecked();
    expect(screen.getByText(strings.preflight.recommendedBecause)).toBeInTheDocument();
    const launch = screen.getByRole("button", { name: strings.launch.launch });
    await waitFor(() => expect(launch).toBeEnabled(), { timeout: 3000 });
    await tap(launch);
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(strings.launch.willSupersede)).toBeInTheDocument();
    expect(writes(calls)).toHaveLength(0);
    await tap(within(dialog).getByRole("button", { name: strings.launch.launch }));
    await waitFor(() => expect(writes(calls).map((c) => c.path)).toEqual([`${LAUNCH}/Build`, `${LAUNCH}/Launch`]));
    expect(taps).toBe(5);
    const build = writes(calls)[0]?.body as { specJson: string; resolutions: Resolutions };
    expect(JSON.parse(build.specJson)).toMatchObject({ topics: ["a", "b", "c"], stance_quotes: [WIRE.suggestedQuote] });
    expect(build.resolutions).toEqual([{ commissionId: "p-4", choice: "RESOLUTION_SUPERSEDE" }]);
  });

  it("supersedes both paused commissions by default and sends both choices with Build (rulings P2)", async () => {
    const user = userEvent.setup();
    const three = { topics: ["a", "b", "c"].map((name) => ({ ...topics.topics[0], topic: name, title: { value: `Topic ${name}` }, liveCommissionIds: [] })) };
    const calls = stubWire((path, body) => {
      if (path === `${LAUNCH}/GetLaunchOptions`) return { ...options, stanceQuotes: [{ text: WIRE.suggestedQuote, source: WIRE.quoteSource, commissionId: "old-1" }] };
      if (path === `${ACTIVITY}/ListTopics`) return three;
      if (path === `${LAUNCH}/Preflight`) {
        const both = [verdict("a", "p-3", "paused", body, "RESOLUTION_SUPERSEDE", WIRE.supersedeReason), verdict("b", "p-4", "paused", body, "RESOLUTION_SUPERSEDE", WIRE.supersedeReason)];
        const open = both.filter((v) => v.blocks);
        return { ...preflight(true), topics: both, launchable: open.length === 0, blockers: open.map((v) => `${v.topic}: ${v.commissionId} ${v.message}`) };
      }
      if (path === `${LAUNCH}/Build`) return { packet: { ok: true, commissionId: "a-w1-20261112" } };
      return {};
    });
    render("/launch");
    let taps = 0;
    const tap = async (el: HTMLElement) => {
      taps += 1;
      await user.click(el);
    };
    for (const name of WIRE.threeTopics) await tap(await screen.findByRole("checkbox", { name }));
    const rows = await screen.findAllByTestId(selectors.launch.blocker, {}, { timeout: 3000 });
    expect(rows).toHaveLength(2);
    for (const row of rows) await waitFor(() => expect(within(row).getByRole("radio", { name: new RegExp(strings.preflight.choiceSupersede) })).toBeChecked());
    const launch = screen.getByRole("button", { name: strings.launch.launch });
    await waitFor(() => expect(launch).toBeEnabled(), { timeout: 3000 });
    await tap(launch);
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getAllByText(strings.launch.willSupersede)).toHaveLength(2);
    await tap(within(dialog).getByRole("button", { name: strings.launch.launch }));
    await waitFor(() => expect(writes(calls).map((c) => c.path)).toEqual([`${LAUNCH}/Build`, `${LAUNCH}/Launch`]));
    expect(taps).toBe(5);
    expect((writes(calls)[0]?.body as { resolutions: Resolutions }).resolutions).toEqual([
      { commissionId: "p-3", choice: "RESOLUTION_SUPERSEDE" },
      { commissionId: "p-4", choice: "RESOLUTION_SUPERSEDE" },
    ]);
  });

  it("keeps the form when the operator leaves and returns, until it is cleared (rulings V1)", async () => {
    const user = userEvent.setup();
    stubWire((path, body) => (path === `${LAUNCH}/GetLaunchOptions` ? options : path === `${ACTIVITY}/ListTopics` ? topics : path === `${LAUNCH}/Preflight` ? preflight(true, body) : {}));
    render("/launch");
    await fill(user);
    await user.click(await screen.findByRole("radio", { name: strings.preflight.choiceAlongside }, { timeout: 3000 }));
    cleanup();
    render("/launch");
    expect(await screen.findByRole("checkbox", { name: /Local models/ })).toBeChecked();
    expect(screen.getByLabelText(strings.launch.quoteN)).toHaveValue("make it reliable");
    expect(await screen.findByRole("radio", { name: strings.preflight.choiceAlongside }, { timeout: 3000 })).toBeChecked();
    await user.click(screen.getByRole("button", { name: strings.launch.startOver }));
    expect(screen.getByLabelText(strings.launch.quoteN)).toHaveValue("");
  });

  it("ends in a manual start when automatic launch is unavailable", async () => {
    stubWire((path) => (path === `${LAUNCH}/GetLaunchOptions` ? { ...options, automaticAvailable: false, automaticReason: "web-console down" } : path === `${ACTIVITY}/ListTopics` ? topics : {}));
    render("/launch");
    expect(await screen.findByText(strings.launch.manualModeTitle)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.launch.launch })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.launch.buildManual })).toBeDisabled();
  });
});

describe("Launch status", () => {
  const prepared = (state: string) => ({ commission: { id: "p1", folder: "/home/u/.vrooli/research/cohorts/p1", state, title: { value: "Packet one" }, stateEvidence: [{ known: true, reason: WIRE.uncertainReason }] } });
  const messages = { stance: "Stance. Reply \"ready\" and wait", kernel: WIRE.kernelGoal, charterPath: "/home/u/.vrooli/research/cohorts/p1/CHARTER.md" };
  const check = { packet: { ok: true, gates: [{ name: "kernel_chars", value: "480", limit: "600", ok: true }] }, topics: [], launchable: true };

  it("launches a prepared packet as it is only after confirming, with the manual messages beside it (a read, rulings P4)", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) =>
      path === `${ACTIVITY}/GetCommission`
        ? prepared("COMMISSION_STATE_PREPARED")
        : path === `${LAUNCH}/GetLaunchOptions`
          ? { automaticAvailable: true }
          : path === `${LAUNCH}/Preflight`
            ? check
            : path === `${LAUNCH}/GetLaunchMessages`
              ? messages
              : {},
    );
    render("/launch/p1");
    expect(await screen.findByText(WIRE.kernelGoal, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(strings.preflight.noEarlier)).toBeInTheDocument();
    const launch = screen.getByRole("button", { name: strings.launch.launch });
    await waitFor(() => expect(launch).toBeEnabled());
    await user.click(launch);
    const dialog = await screen.findByRole("alertdialog");
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.launch.launch }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Launch`, body: { commissionId: "p1" } }]));
  });

  it("says why an unknown packet cannot be launched on the operator's word, hides its two messages and offers a relaunch as a new commission (rulings V1, P1)", async () => {
    const calls = stubWire((path) =>
      path === `${ACTIVITY}/GetCommission`
        ? prepared("COMMISSION_STATE_UNKNOWN")
        : path === `${LAUNCH}/GetLaunchOptions`
          ? { automaticAvailable: false, automaticReason: "web-console down" }
          : path === `${LAUNCH}/Preflight`
            ? { ...check, launchable: false, blockers: [WIRE.unknownBlocker], neverStarted: { assertable: false, reason: WIRE.neverRefused } }
            : {},
    );
    render("/launch/p1");
    expect(await screen.findByText(WIRE.uncertainReason)).toBeInTheDocument();
    expect(await screen.findByText(strings.launch.manualModeTitle)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.launch.launch })).not.toBeInTheDocument();
    expect(await screen.findByText(WIRE.neverRefused)).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: strings.launchStatus.neverStarted })).not.toBeInTheDocument();
    expect(screen.queryByText(strings.launchStatus.manualTitle)).not.toBeInTheDocument();
    // The API's line, without the command it quotes.
    expect(await screen.findByText(WIRE.unknownBlockerWords)).toBeInTheDocument();
    expect(calls.some((c) => c.path === `${LAUNCH}/Launch` || c.path === `${LAUNCH}/GetLaunchMessages`)).toBe(false);
    expect(screen.getAllByRole("button", { name: strings.launchStatus.relaunchNew }).length).toBeGreaterThan(0);
  });

  it("launches an unknown packet once the operator says it never started, and records that they said so (rulings P1)", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path, body) => {
      if (path === `${ACTIVITY}/GetCommission`) return prepared("COMMISSION_STATE_UNKNOWN");
      if (path === `${LAUNCH}/GetLaunchOptions`) return { automaticAvailable: true };
      if (path === `${LAUNCH}/GetLaunchMessages`) return messages;
      if (path === `${LAUNCH}/Preflight`) {
        const asserted = (body as { assertNeverStarted?: boolean }).assertNeverStarted === true;
        return { ...check, launchable: asserted, blockers: asserted ? [] : ["p1 is unknown (no session); launch it only if you know it was never started"], neverStarted: { assertable: true, reason: WIRE.neverReason } };
      }
      return {};
    });
    render("/launch/p1");
    const never = await screen.findByRole("checkbox", { name: strings.launchStatus.neverStarted });
    expect(screen.getByText(WIRE.neverReason)).toBeInTheDocument();
    // Until the operator says so, nothing can start: no Launch, no messages to paste, and the API's line is left to the choice.
    expect(screen.getByRole("button", { name: strings.launch.launch })).toBeDisabled();
    expect(screen.queryByText(strings.launchStatus.manualTitle)).not.toBeInTheDocument();
    expect(screen.queryByText(strings.preflight.blocked)).not.toBeInTheDocument();
    await user.click(never);
    const launch = screen.getByRole("button", { name: strings.launch.launch });
    await waitFor(() => expect(launch).toBeEnabled());
    expect(await screen.findByText(WIRE.kernelGoal, { exact: false })).toBeInTheDocument();
    await user.click(launch);
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(strings.launchStatus.assertRecorded)).toBeInTheDocument();
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.launch.launch }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Launch`, body: { commissionId: "p1", assertNeverStarted: true } }]));
  });

  it("shows each step of a start in progress", async () => {
    stubWire((path) =>
      path === `${ACTIVITY}/GetCommission`
        ? { commission: { id: "p1", state: "COMMISSION_STATE_LIVE" } }
        : path === `${LAUNCH}/GetLaunch`
          ? { launch: { running: true, webConsoleSessionId: WIRE.session, steps: [{ step: "created", at: "2026-11-11T09:30:00Z" }] } }
          : {},
    );
    render("/launch/p1");
    expect(await screen.findByText(strings.launchStatus.running)).toBeInTheDocument();
    expect(screen.getByText(WIRE.session)).toBeInTheDocument();
    expect(screen.getByText(strings.launchStatus.stepNow)).toBeInTheDocument();
  });

  it("relaunches a failed start from the same packet only after confirming", async () => {
    const user = userEvent.setup();
    const calls = stubWire((path) =>
      path === `${ACTIVITY}/GetCommission`
        ? { commission: { id: "p1", state: "COMMISSION_STATE_STOPPED" } }
        : path === `${LAUNCH}/GetLaunch`
          ? { launch: { failedStep: "ready_seen", failedReason: "no ready within 120 s", steps: [{ step: "created" }, { step: "composer_ready" }, { step: "stance_sent" }] } }
          : path === `${LAUNCH}/Build`
            ? { packet: { ok: true, commissionId: WIRE.relaunchId } }
            : {},
    );
    render("/launch/p1");
    expect(await screen.findByText(strings.launch.failedAt)).toBeInTheDocument();
    expect(screen.getByText(strings.launchStatus.stepFailed)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: strings.launchStatus.relaunchNew }));
    const dialog = await screen.findByRole("alertdialog");
    expect(writes(calls)).toHaveLength(0);
    await user.click(within(dialog).getByRole("button", { name: strings.launchStatus.relaunchBuild }));
    await waitFor(() => expect(writes(calls)).toEqual([{ path: `${LAUNCH}/Build`, body: { relaunchOf: "p1" } }]));
    expect((await screen.findAllByText(WIRE.relaunchId)).length).toBeGreaterThan(0);
  });

  it("points an ended commission back to its page", async () => {
    stubWire((path) => (path === `${ACTIVITY}/GetCommission` ? { commission: { id: "p1", state: "COMMISSION_STATE_ENDED", session: { webConsoleSessionId: "s1" } } } : {}));
    render("/launch/p1");
    expect(await screen.findByText(strings.launchStatus.nothingHere)).toBeInTheDocument();
  });
});
