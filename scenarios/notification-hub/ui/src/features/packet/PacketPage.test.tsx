import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { create, toJson } from "@bufbuild/protobuf";
import { AskSchema, AskAttachmentNoteSchema, AskContextRequestSchema, AskQuestionResponseSchema, AskRevisionSchema, AskOptionSchema } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { fetchAttachment, resolveLinkedArtifact } from "../../api/attachments";
import { conversationsClient, identityClient } from "../../api/notifications";
import { TestAppRouter } from "../../app/routes";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { ar, en } from "../../i18n/catalog";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { makeAsk, makeAttachment, makeQuestion } from "./mocks/askFactory";
import { emptyDraft, withAttachmentNote } from "./model/draft";
import { clearAskDrafts, readAskAttempt, readAskDraft, writeAskDraft } from "./model/draftStore";

vi.mock("../../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(), login: vi.fn() },
  conversationsClient: { getAsk: vi.fn(), answerRootWithPassword: vi.fn(), answer: vi.fn(), dismiss: vi.fn(), listAsks: vi.fn(), requestContext: vi.fn() },
}));
vi.mock("../../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/client")>()),
}));
vi.mock("../../api/attachments", () => ({
  attachmentUrl: (askId: string, name: string, path = "") => `/files/${askId}/${name}${path ? `?path=${path}` : ""}`,
  fetchAttachment: vi.fn(),
  resolveLinkedArtifact: vi.fn((askId: string, path: string) => Promise.resolve({ name: path.split("/").at(-1) ?? "file.md", kind: "markdown", mimeType: "text/markdown", sizeBytes: 12, contentUrl: `/artifact?ask=${askId}` })),
}));
// The viewer has its own tests; here it only has to say what it was asked to open.
vi.mock("./viewer/AttachmentViewer", () => ({
  AttachmentViewer: ({ target, onClose }: { target: { name: string; path: string }; onClose: () => void }) => <div data-testid="viewer-stub">{`${target.name}|${target.path}`}<button type="button" onClick={onClose}>Close viewer</button></div>,
}));

const TEXT_BACKUPS_ARE_BLOCKED = "Backups are blocked.";
const TEXT_BACKUPS_HAVE_BEEN_BLOCKED = "Backups have been blocked for an hour.";
const TEXT_BACKUPS_SUCCEEDED_AGAIN = "Backups succeeded again.";
const TEXT_DOES_THE_GATE_PASS = "Does the gate pass?";
const TEXT_PRIVATE_DRAFT_NOTE = "private draft note";
const TEXT_REPLACED_BY_A_NEWER = "Replaced by a newer version of this decision";
const TEXT_SET_B = "Set B";
const TEXT_THIS_KEEPS_THE_QUALITY = "This keeps the quality goal useful.";
const TEXT_TWO_REQUIRED_PLATFORM_CHECKS = "Two required platform checks still fail.";
const TEXT_WHAT_DOES_THE_GATE = "What does the gate mean?";

const RAW_URGENCY = "high";
const SOURCE_TOKEN = "operator";
const SENDER_NAME = "Research Desk";
const ASK_ID = "ask-1";
const LATEST_ANSWER = "Second answer";
const GATE_QUESTION = "What gate?";
const BUILD_QUESTION = "Which build?";
const REVIEW_STEP = "The review step.";
const SHARED_ANSWER = "One reply for both.";
const CARRIED_ANSWER = "Carried answer";
const FRESH_ANSWER = "Fresh answer";
const SET_PROMPT = "Which studio set?";
const WHY = "Why it matters to the release";

const text = en.packet;
const SIGNED_IN = { signedIn: true, subject: "owner-1", email: "owner@example.test" };

function renderAsk(askId = "ask-1", from?: string) {
  const entry = from ? { pathname: `/asks/${askId}`, state: { from } } : `/asks/${askId}`;
  return renderWithProviders(<TestAppRouter initialEntries={[entry]} />, { withoutRouter: true });
}

const optionByKey = async (key: string) => (await screen.findAllByTestId(selectors.asks.option)).find((option) => option.getAttribute("data-option-key") === key) as HTMLElement;
const send = () => screen.getByRole("button", { name: text.form.submit });
const openNote = (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByText(text.form.noteLabel, { selector: "summary" }));

function setPhone(isPhone: boolean) {
  vi.spyOn(window, "matchMedia").mockImplementation(
    (media) =>
      ({ media, matches: media.includes("max-width") ? isPhone : !isPhone, onchange: null, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {}, dispatchEvent: () => false }) as MediaQueryList,
  );
}

beforeEach(async () => {
  await setLocale("en");
  vi.mocked(identityClient.getSession).mockResolvedValue(SIGNED_IN as never);
  vi.mocked(conversationsClient.listAsks).mockResolvedValue({ asks: [], totalAwaiting: 0 } as never);
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.clearAllMocks();
  clearAskDrafts();
});


describe("root reauthentication entry", () => {
  it("opens a password-read root decision and offers a fresh answer verification", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(conversationsClient.getAsk).mockRejectedValue(new ConnectError("sign in", Code.Unauthenticated));
    const decision = makeAsk({ id: "ask-1", source: "root-rebaseline", sourceEventType: "root.rebaseline.decision_requested.v1" });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ mfaRequired: false, ask: toJson(AskSchema, decision) }))));
    renderAsk();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: text.rootRead.action }));
    const form = screen.getByRole("heading", { name: text.rootRead.title }).closest("form")!;
    await user.type(within(form).getByLabelText(en.session.email), "owner@example.test");
    await user.type(within(form).getByLabelText(en.session.password), "secret");
    await user.click(within(form).getByRole("button", { name: text.rootRead.action }));
    await user.click(await screen.findByRole("button", { name: text.rootPassword.openForm }));
    expect(screen.getByRole("heading", { name: text.rootPassword.title })).toBeInTheDocument();
    expect(within(screen.getByRole("heading", { name: text.rootPassword.title }).closest("form")!).getByLabelText(text.rootPassword.password)).toHaveValue("");
    vi.unstubAllGlobals();
  });
  it.each(["recorded", "changed", "uncertain"] as const)("handles a %s password answer without leaving a stale decision actionable", async (outcome) => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(conversationsClient.getAsk).mockRejectedValue(new ConnectError("sign in", Code.Unauthenticated));
    const decision = makeAsk({ id: "ask-1", source: "root-rebaseline", sourceEventType: "root.rebaseline.decision_requested.v1", options: [create(AskOptionSchema, { key: "approve", label: "Approve" })] });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ mfaRequired: false, ask: toJson(AskSchema, decision) }))));
    if (outcome === "recorded") vi.mocked(conversationsClient.answerRootWithPassword).mockResolvedValue({ answeredAt: "now" } as never);
    else vi.mocked(conversationsClient.answerRootWithPassword).mockRejectedValue(outcome === "changed" ? new ConnectError("changed", Code.FailedPrecondition) : new Error("lost response"));
    renderAsk(); const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: text.rootRead.action }));
    const readForm = screen.getByRole("heading", { name: text.rootRead.title }).closest("form")!;
    await user.type(within(readForm).getByLabelText(en.session.email), "owner@example.test");
    await user.type(within(readForm).getByLabelText(en.session.password), "secret");
    await user.click(within(readForm).getByRole("button", { name: text.rootRead.action }));
    await user.click(await screen.findByRole("button", { name: text.rootPassword.openForm }));
    const answerForm = screen.getByRole("heading", { name: text.rootPassword.title }).closest("form")!;
    await user.click(within(answerForm).getByRole("radio", { name: "Approve" }));
    await user.type(within(answerForm).getByLabelText(text.rootPassword.email), "owner@example.test");
    await user.type(within(answerForm).getByLabelText(text.rootPassword.password), "secret");
    await user.click(within(answerForm).getByRole("button", { name: text.rootPassword.verify }));
    if (outcome === "recorded") {
      expect(await screen.findByRole("status")).toHaveTextContent(text.rootPassword.recorded);
      expect(screen.queryByRole("button", { name: text.rootRead.action })).not.toBeInTheDocument();
    } else {
      expect(await screen.findByRole("alert")).toHaveTextContent(outcome === "changed" ? text.rootPassword.changed : text.rootPassword.uncertain);
      expect(screen.getByRole("button", { name: text.rootRead.action })).toBeInTheDocument();
    }
    expect(screen.queryByRole("button", { name: text.rootPassword.openForm })).not.toBeInTheDocument();
    vi.unstubAllGlobals();
  });
  it("opens a protected decision after cold sign-in and MFA", async () => {
    let signedIn = false;
    vi.mocked(identityClient.getSession).mockImplementation(() => Promise.resolve((signedIn ? SIGNED_IN : { signedIn: false }) as never));
    vi.mocked(identityClient.login).mockImplementation((request) => {
      if (!request.mfaChallenge) return Promise.resolve({ mfaRequired: true, mfaChallenge: "challenge" } as never);
      signedIn = true;
      return Promise.resolve({ mfaRequired: false } as never);
    });
    vi.mocked(conversationsClient.getAsk).mockImplementation(() => {
      if (!signedIn) return Promise.reject(new ConnectError("sign in", Code.Unauthenticated));
      return Promise.resolve({ ask: makeAsk({ source: "root-rebaseline", sourceEventType: "root.rebaseline.decision_requested.v1" }) } as never);
    });
    renderAsk();
    const user = userEvent.setup();
    await user.type(await screen.findByTestId(selectors.session.email, {}, { timeout: 8000 }), "owner@example.test");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));
    await user.type(await screen.findByLabelText(en.session.totpCode), "123456");
    await user.click(screen.getByTestId(selectors.session.submit));
    expect(await screen.findByRole("button", { name: text.rootPassword.openForm })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.question)).toBeInTheDocument();
  }, 15000);

  it("offers password verification only for the root decision and keeps the normal answer available", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ source: "root-rebaseline", sourceEventType: "root.rebaseline.decision_requested.v1" }) } as never);
    renderAsk();
    const entry = await screen.findByRole("button", { name: text.rootPassword.openForm });
    expect(send()).toBeInTheDocument();
    await userEvent.click(entry);
    expect(screen.getByRole("heading", { name: text.rootPassword.title })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: text.form.submit })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: text.rootPassword.cancel }));
    expect(send()).toBeInTheDocument();
  });
});

describe("a plain decision", () => {
  it("allows recipient-scoped reads without a session but requires one to answer", async () => {
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: false } as never);
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(conversationsClient.getAsk).toHaveBeenCalledWith({ askId: "ask-1" });
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
  });

  it("does not retry a recipient-scoped NotFound or disclose whether the ask exists", async () => {
    vi.mocked(conversationsClient.getAsk).mockRejectedValue(new ConnectError("not found", Code.NotFound));
    renderAsk();

    expect(await screen.findByRole("heading", { name: text.status.notFound })).toBeInTheDocument();
    // The shell's tab bar has a link of the same name; the way back is the one inside the page.
    const page = screen.getByRole("region", { name: text.status.notFound });
    expect(within(page).getByRole("link", { name: en.layout.nav.asks })).toHaveAttribute("href", "/asks");
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(1);
  });

  it("says a transient load failure is only that, keeps the decision on offer, and loads it on retry", async () => {
    vi.mocked(conversationsClient.getAsk).mockRejectedValue(new ConnectError("unavailable", Code.Unavailable));
    const user = userEvent.setup();
    renderAsk();

    // The query retries twice before giving up; the failure then names itself and is not a "not found".
    const alert = await screen.findByRole("alert", undefined, { timeout: 8000 });
    expect(alert).toHaveTextContent(text.status.loadFailed);
    expect(alert).toHaveTextContent(text.status.loadFailedDetail);
    expect(screen.queryByText(text.status.notFound)).not.toBeInTheDocument();
    const page = screen.getByRole("region", { name: text.status.loadFailed });
    expect(within(page).getByRole("link", { name: en.layout.nav.asks })).toHaveAttribute("href", "/asks");

    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    await user.click(screen.getByTestId(selectors.packet.loadRetry));
    expect(await screen.findByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  }, 15000);

  it("lets the requester's own text take its own direction, so a page in Arabic still reads an English question correctly", async () => {
    await setLocale("ar");
    const bigPicture = "Why it matters";
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ recommended: "a", bigPicture }) } as never);
    renderAsk();

    const question = await screen.findByTestId(selectors.asks.question);
    expect(document.documentElement).toHaveAttribute("dir", "rtl");
    expect(question).toHaveAttribute("dir", "auto");
    expect(screen.getByText(bigPicture)).toHaveAttribute("dir", "auto");
  });

  it("names the urgency in the page's language and never shows the raw token", async () => {
    await setLocale("ar");
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ urgency: RAW_URGENCY }) } as never);
    renderAsk();

    await screen.findByTestId(selectors.asks.question);
    expect(screen.getByText(ar.asks.urgency.high)).toBeInTheDocument();
    expect(screen.queryByText(RAW_URGENCY)).not.toBeInTheDocument();
  });

  it("keeps the header row for navigation and puts the request id on its own quiet line", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    renderAsk();

    await screen.findByTestId(selectors.asks.question);
    const id = screen.getByText(ASK_ID);
    expect(id.closest("p")).toHaveTextContent(`${text.status.requestId} ${ASK_ID}`);
    const page = id.closest("section")!;
    const back = within(page).getByRole("link", { name: en.layout.nav.asks });
    expect(page.firstElementChild).toContainElement(back);
    // The id is a footnote after the question, never ahead of it.
    expect(screen.getByTestId(selectors.asks.question).compareDocumentPosition(id) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  // The page showed `ask.source` ("operator") where the sender's name belongs.
  it("names the sender, and falls back to the source only when it gave no name", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ source: SOURCE_TOKEN, sender: SENDER_NAME }) } as never);
    const { unmount } = renderAsk();
    expect(await screen.findByText(SENDER_NAME)).toBeInTheDocument();
    expect(screen.queryByText(SOURCE_TOKEN)).not.toBeInTheDocument();
    unmount();

    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ source: SOURCE_TOKEN, sender: "" }) } as never);
    renderAsk();
    expect(await screen.findByText(SOURCE_TOKEN)).toBeInTheDocument();
  });

  it("shows the recommended option first, marked, with the default's countdown", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ defaultEligibleAt: new Date(Date.now() + 3 * 3_600_000 + 30_000).toISOString() }) } as never);
    renderAsk();

    const options = await screen.findAllByTestId(selectors.asks.option);
    expect(options.map((option) => option.getAttribute("data-option-key"))).toEqual(["re-aim", "close", "other"]);
    expect(options.every((option) => !(option as HTMLInputElement).checked)).toBe(true);
    expect(within(options[0]!.parentElement!).getByText(text.form.recommended)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.defaultNotice)).toHaveTextContent("Re-aim at quality applies in 3 h");
    expect(send()).toBeDisabled();
  });

  it("lists options in the requester's order, marks none and can still send, when no option is recommended", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ recommended: "", recommendationReason: "", defaultAnswer: "" }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    const options = await screen.findAllByTestId(selectors.asks.option);
    expect(options.map((option) => option.getAttribute("data-option-key"))).toEqual(["close", "re-aim", "other"]);
    expect(screen.queryByText(text.form.recommended)).not.toBeInTheDocument();
    await user.click(await optionByKey("close"));
    expect(send()).toBeEnabled();
  });

  it("always offers Other, even when the requester's options omit it", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ options: [{ key: "yes", label: "Yes" }, { key: "no", label: "No" }] }) } as never);
    renderAsk();
    expect(await optionByKey("other")).toBeInTheDocument();
  });

  it("submits an Other answer with free-form text, then shows the recorded decision", async () => {
    const user = userEvent.setup();
    const ask = makeAsk();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "other", answerLabel: "Other", note: "Keep it open and narrow the scope." }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({ answer: "other" } as never);
    renderAsk();

    await user.click(await optionByKey("other"));
    expect(send()).toBeDisabled();
    await user.type(screen.getByLabelText(text.form.otherPrompt), "Keep it open and narrow the scope.");
    await user.click(send());

    await waitFor(() =>
      expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "other", note: "Keep it open and narrow the scope.", responses: [], attachmentNotes: [], changes: {}, images: [] }),
    );
    expect(await screen.findByText(text.outcome.recorded.replace("{{option}}", "Other"))).toBeInTheDocument();
  });

  it("lets the operator dismiss an open decision without recording an answer", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "dismissed", reason: "dismissed by recipient: not applicable" }) } as never);
    vi.mocked(conversationsClient.dismiss).mockResolvedValue({ state: "dismissed" } as never);
    renderAsk();

    await user.click(await screen.findByRole("button", { name: text.form.dismiss }));

    await waitFor(() => expect(conversationsClient.dismiss).toHaveBeenCalledWith({ askId: "ask-1", note: "" }));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(await screen.findByText(text.status.dismissed.replace("{{reason}}", "dismissed by recipient: not applicable"))).toBeInTheDocument();
  });

  it("is accessible by role and label", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    const view = renderAsk();

    expect(await screen.findAllByRole("radio")).toHaveLength(3);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(makeAsk().question);
    expect(send()).toBeDisabled();
    await expectNoA11yViolations(view.container);
  });

  it("shows the decision's purpose at once and keeps technical context optional", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ bigPicture: "This keeps the quality goal useful.", technicalDetails: "Two required platform checks still fail." }) } as never);
    renderAsk();

    expect((await screen.findByText(TEXT_THIS_KEEPS_THE_QUALITY)).closest("details")).toHaveProperty("open", true);
    expect(screen.getByText(TEXT_TWO_REQUIRED_PLATFORM_CHECKS).closest("details")).toHaveProperty("open", false);
  });

  it("preserves the note while the context disclosures open and close", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ bigPicture: "Purpose.", technicalDetails: "Details." }) } as never);
    renderAsk();

    await user.click(await screen.findByText(text.form.noteLabel, { selector: "summary" }));
    const note = screen.getByTestId(selectors.asks.note);
    await user.type(note, "keep the macOS follow-up");
    for (const label of [text.cover.bigPicture, text.cover.technicalDetails]) {
      await user.click(screen.getByText(label));
      expect(note).toHaveValue("keep the macOS follow-up");
      await user.click(screen.getByText(label));
    }
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });
});

describe("what protects the operator's answer", () => {
  beforeEach(() => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
  });

  it("keeps the unsent draft when the session expires, and hides the controls", async () => {
    const user = userEvent.setup();
    renderAsk();
    await user.click(await optionByKey("close"));
    await openNote(user);
    await user.type(screen.getByTestId(selectors.asks.note), "keep this draft");

    window.dispatchEvent(new Event("notification-hub:session-expired"));

    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(readAskDraft("owner-1", "ask-1")).toMatchObject({ answer: "close", note: "keep this draft" });
  });

  it("restores the subject-scoped draft after a reload", async () => {
    const user = userEvent.setup();
    const first = renderAsk();
    await user.click(await optionByKey("close"));
    await openNote(user);
    await user.type(screen.getByTestId(selectors.asks.note), "resume after refresh");
    first.unmount();

    renderAsk();
    expect(await optionByKey("close")).toBeChecked();
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("resume after refresh");
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("keeps each owner's draft private when the verified identity changes", async () => {
    const user = userEvent.setup();
    const view = renderAsk();
    await user.click(await optionByKey("close"));

    view.queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-2", email: "other@example.test" });
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    await waitFor(async () => expect(await optionByKey("close")).not.toBeChecked());
    expect(readAskDraft("owner-1", "ask-1").answer).toBe("close");
    expect(readAskDraft("owner-2", "ask-1").answer).toBe("");
  });

  it("requires a deliberate choice and send, then confirms from the authoritative readback", async () => {
    const user = userEvent.setup();
    let finishReadback!: (response: unknown) => void;
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockImplementationOnce(() => new Promise((resolve) => { finishReadback = resolve; }) as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    await user.click(send());

    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    expect(screen.queryByText(text.outcome.recorded.replace("{{option}}", "Close the goal"))).not.toBeInTheDocument();
    expect(screen.getAllByText(text.outcome.checking)).toHaveLength(2);
    finishReadback({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) });
    expect(await screen.findByText(text.outcome.recorded.replace("{{option}}", "Close the goal"))).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("freezes the attempt and never replays it after an uncertain response", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();

    await user.click(await optionByKey("close"));
    await openNote(user);
    const note = screen.getByTestId(selectors.asks.note);
    await user.type(note, "keep this attempt");
    await user.click(send());
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();
    expect(note).toHaveValue("keep this attempt");
    expect(note).toBeDisabled();
    expect(screen.queryByRole("button", { name: text.form.submit })).not.toBeInTheDocument();
    expect(readAskAttempt("owner-1", "ask-1")).toEqual({ answer: "close", note: "keep this attempt", subject: "owner-1", responses: [], attachmentNotes: [], changes: {}, images: [] });

    await user.click(screen.getByRole("button", { name: text.outcome.checkStatus }));
    expect(await screen.findByText(text.outcome.stillOpen)).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2);
  });

  it("reloads a decision the requester revised while answering, keeps the draft and allows sending again", async () => {
    const user = userEvent.setup();
    const revised = makeAsk({ question: "Close the goal now that the gate changed?", revisionCount: 1 });
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: revised } as never);
    vi.mocked(conversationsClient.answer)
      .mockRejectedValueOnce(new ConnectError("ask_changed", Code.FailedPrecondition, new Headers({ "Hub-Error-Reason": "ask_changed" })))
      .mockResolvedValueOnce({} as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    await openNote(user);
    await user.type(screen.getByTestId(selectors.asks.note), "still my view");
    await user.click(send());

    expect(await screen.findByText(text.outcome.changed)).toBeInTheDocument();
    expect(await screen.findByText(revised.question)).toBeInTheDocument();
    expect(screen.queryByText(text.outcome.uncertain)).not.toBeInTheDocument();
    expect(readAskAttempt("owner-1", "ask-1")).toBeNull();
    expect(readAskDraft("owner-1", "ask-1")).toMatchObject({ answer: "close", note: "still my view" });
    expect(screen.getByTestId(selectors.asks.note)).toHaveValue("still my view");

    await user.click(send());
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(2));
    expect(screen.queryByText(text.outcome.changed)).not.toBeInTheDocument();
  });

  it("confirms an uncertain answer from readback without replaying the mutation", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();

    await user.click(await optionByKey("close"));
    await user.click(send());
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: text.outcome.checkStatus }));

    expect(await screen.findByText(text.outcome.recorded.replace("{{option}}", "Close the goal"))).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("reports a different recorded result as a conflict", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "re-aim", answerLabel: "Re-aim at quality" }) } as never);
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();

    await user.click(await optionByKey("close"));
    await user.click(send());
    await user.click(await screen.findByRole("button", { name: text.outcome.checkStatus }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Re-aim at quality");
    expect(screen.getByRole("alert")).toHaveTextContent("Close the goal");
  });

  it("keeps status readback after session expiry without restoring answer controls", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    renderAsk();
    await user.click(await optionByKey("close"));
    await openNote(user);
    await user.type(screen.getByTestId(selectors.asks.note), "private draft note");
    await user.click(send());
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();

    window.dispatchEvent(new Event("notification-hub:session-expired"));
    expect(await screen.findByTestId(selectors.session.form)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.asks.option)).not.toBeInTheDocument();
    expect(screen.queryByText(TEXT_PRIVATE_DRAFT_NOTE)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: text.outcome.checkStatus }));
    expect(await screen.findByText(text.outcome.stillOpen)).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("restores an uncertain attempt after a refresh without replaying it", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    const first = renderAsk();
    await user.click(await optionByKey("close"));
    await user.click(send());
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();
    first.unmount();

    renderAsk();
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: text.outcome.checkStatus })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: text.form.submit })).not.toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("keeps an uncertain attempt private after the verified identity changes", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockRejectedValueOnce(new Error("connection lost"));
    const view = renderAsk();
    await user.click(await optionByKey("close"));
    await user.click(send());
    expect(await screen.findByText(text.outcome.uncertain)).toBeInTheDocument();

    view.queryClient.setQueryData(["owner-session"], { signedIn: true, subject: "owner-2", email: "other@example.test" });
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledTimes(2));
    expect(await screen.findAllByTestId(selectors.asks.option)).toHaveLength(3);
    expect(screen.queryByText(text.outcome.uncertain)).not.toBeInTheDocument();
    expect(readAskAttempt("owner-2", "ask-1")).toBeNull();
    expect(readAskAttempt("owner-1", "ask-1")?.answer).toBe("close");
  });

  it("keeps one pending attempt locked through an open read and confirms a delayed commit", async () => {
    const user = userEvent.setup();
    let finishAnswer!: (value: unknown) => void;
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    vi.mocked(conversationsClient.answer).mockImplementation(() => new Promise((resolve) => { finishAnswer = resolve; }) as never);
    renderAsk();
    await user.click(await optionByKey("close"));
    await user.click(send());
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));

    await user.click(screen.getByRole("button", { name: text.outcome.checkStatus }));
    expect(await screen.findByText(text.outcome.stillOpen)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: text.form.submit })).not.toBeInTheDocument();

    finishAnswer({});
    expect(await screen.findByText(text.outcome.recorded.replace("{{option}}", "Close the goal"))).toBeInTheDocument();
    expect(conversationsClient.answer).toHaveBeenCalledTimes(1);
  });

  it("offers the next waiting decision once an answer is recorded", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [makeAsk({ id: "ask-1" }), makeAsk({ id: "ask-2", question: "Waiting on the agent", openContextRequestCount: 1 }), makeAsk({ id: "ask-3", title: "Ship the release?" })],
      totalAwaiting: 3,
    } as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    await user.click(send());

    const next = await screen.findByTestId(selectors.packet.nextDecision);
    expect(next).toHaveAttribute("href", "/asks/ask-3");
    expect(next).toHaveTextContent("Ship the release?");
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ limit: 25, stateFilter: "awaiting" });
  });

  it("says when nothing else is waiting", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValueOnce({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    await user.click(send());

    expect(await screen.findByText(text.outcome.allDone)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.nextDecision)).not.toBeInTheDocument();
  });
});

describe("decision states", () => {
  it("draws a question with nothing attached as just the cover and the form, with no empty sections", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ attachments: [], questions: [] }) } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.question)).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.packet.form)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.attachments)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.explainer)).not.toBeInTheDocument();
  });

  it("shows a withdrawn decision without answer controls", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "withdrawn" }) } as never);
    renderAsk();

    expect(await screen.findByText(text.status.withdrawn)).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
    expect(screen.queryByRole("button", { name: text.form.submit })).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.needContext)).not.toBeInTheDocument();
  });

  it("explains why a decision was closed and links to the one that replaced it", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "withdrawn", reason: "Replaced by a newer version of this decision", supersededByAskId: "ask-2" }) } as never);
    renderAsk();

    expect(await screen.findByText(TEXT_REPLACED_BY_A_NEWER)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: text.status.replacedBy })).toHaveAttribute("href", "/asks/ask-2");
  });

  it("shows the requester's own closing reason without a link when nothing replaced it", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "withdrawn", reason: "Backups succeeded again." }) } as never);
    renderAsk();

    expect(await screen.findByText(TEXT_BACKUPS_SUCCEEDED_AGAIN)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: text.status.replacedBy })).not.toBeInTheDocument();
  });

  it("shows an expired decision as read-only", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "expired" }) } as never);
    renderAsk();

    expect(await screen.findByText(text.status.expired)).toBeInTheDocument();
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
  });

  it("shows an answered decision with the operator's note as a fact, no answer controls, and no repeat of the answer", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal", note: "Keep macOS first." }),
    } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    expect(status).toHaveTextContent("Answered: Close the goal");
    expect(within(status).getByText(text.status.answerNote).nextElementSibling).toHaveTextContent("Keep macOS first.");
    // The answer is said once, in words: never again as "Answer: …", and never with the requester's key.
    expect(status).not.toHaveTextContent("(close)");
    expect(status.textContent.match(/Close the goal/g)).toHaveLength(1);
    expect(screen.queryAllByTestId(selectors.asks.option)).toHaveLength(0);
  });

  it("names the requester's key only when two options share a label", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ state: "answered", answer: "v2", answerLabel: "Approve", options: [{ key: "v1", label: "Approve" }, { key: "v2", label: "Approve" }] }),
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.status)).toHaveTextContent("Answered: Approve (v2)");
  });

  it("shows what was said to each question and the notes on files, in labels, not keys", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        state: "answered",
        answer: "close",
        answerLabel: "Close the goal",
        questions: [makeQuestion({ id: "set", type: "single_choice", prompt: SET_PROMPT, choices: [{ key: "set-b", label: "Open loft" }] })],
        responses: [create(AskQuestionResponseSchema, { questionId: "set", selected: ["set-b"], text: "Brighter please" })],
        attachmentNotes: [create(AskAttachmentNoteSchema, { attachment: "brief.md", note: "Too long" })],
      }),
    } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    expect(within(status).getByText(SET_PROMPT).nextElementSibling).toHaveTextContent("Open loft");
    expect(within(status).getByText(SET_PROMPT).nextElementSibling).toHaveTextContent("Brighter please");
    expect(status).not.toHaveTextContent("set-b");
    expect(within(status).getByText(text.status.answerFileNotes).nextElementSibling).toHaveTextContent("brief.md: Too long");
  });

  it("shows who answered and when as facts under a flat status line, not as a bordered box that looks like a field", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal", answeredBy: "owner@example.test", answeredAt: "2026-11-14T08:00:00Z" }),
    } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    expect(within(status).getByText(text.status.answeredBy).nextElementSibling).toHaveTextContent("owner@example.test");
    expect(within(status).getByText(text.status.answeredAt).nextElementSibling?.querySelector("time")).toHaveAttribute("datetime", "2026-11-14T08:00:00Z");
    expect(status).not.toHaveClass("border");
  });

  it("renders an answer note as safe text and strips non-web link destinations", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ state: "answered", answer: "close", note: "See [the record](https://example.test/record) and [unsafe](javascript:alert(1))." }),
    } as never);
    renderAsk();

    const link = await screen.findByRole("link", { name: "the record" });
    expect(link).toHaveAttribute("href", "https://example.test/record");
    expect(screen.queryByRole("link", { name: "unsafe" })).not.toBeInTheDocument();
  });

  it("allows one deliberate later answer after a default is applied", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "defaulted", answer: "re-aim", defaultAnswer: "re-aim" }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    await user.click(send());
    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith({ askId: "ask-1", answer: "close", note: "", responses: [], attachmentNotes: [], changes: {}, images: [] }));
    expect(screen.queryByTestId(selectors.packet.needContext)).not.toBeInTheDocument();
  });

  it("keeps the original default and its time visible after a later answer", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal", defaultAppliedAt: "2026-10-24T14:10:00Z", late: true }),
    } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    expect(status).toHaveTextContent(text.status.recordedDefault);
    expect(status).toHaveTextContent("Re-aim at quality");
    expect(status).toHaveTextContent(text.status.lateDoesNotUndo);
    expect(status.querySelector("time")).toHaveAttribute("dateTime", "2026-10-24T14:10:00Z");
  });

  it("says how many times a decision was updated and what changed", async () => {
    const options = makeAsk().options;
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        revisionCount: 2,
        revisions: [
          create(AskRevisionSchema, { revision: 2, question: "Backups have been blocked for an hour.", options, revisedAt: "2026-11-13T10:00:00Z" }),
          create(AskRevisionSchema, { revision: 1, question: "Backups are blocked.", options, revisedAt: "2026-11-13T09:00:00Z" }),
        ],
      }),
    } as never);
    renderAsk();

    expect(await screen.findByText(text.revisions.title.replace("{{count}}", "2"))).toBeInTheDocument();
    const second = document.querySelector('[data-revision="2"]') as HTMLElement;
    expect(within(second).getByText(TEXT_BACKUPS_HAVE_BEEN_BLOCKED)).toBeInTheDocument();
    expect(within(second).getByText(makeAsk().question)).toBeInTheDocument();
    const first = document.querySelector('[data-revision="1"]') as HTMLElement;
    expect(within(first).getByText(TEXT_BACKUPS_ARE_BLOCKED)).toBeInTheDocument();
    expect(within(first).getByText(TEXT_BACKUPS_HAVE_BEEN_BLOCKED)).toBeInTheDocument();
  });

  it("says a decision was updated once, and shows no history for one that never changed", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ revisionCount: 1, revisions: [create(AskRevisionSchema, { revision: 1, question: "Earlier.", options: makeAsk().options, revisedAt: "2026-11-13T09:00:00Z" })] }),
    } as never);
    const first = renderAsk();
    expect(await screen.findByText(text.revisions.title_one)).toBeInTheDocument();
    first.unmount();

    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    renderAsk();
    await screen.findAllByRole("radio");
    expect(screen.queryByText(text.revisions.title_one)).not.toBeInTheDocument();
  });
});

describe("where a decision sits in the navigation", () => {
  const tab = (key: "asks" | "history") => screen.getByTestId(`${selectors.layout.navLink({ key })}-tab`);

  it("shows a waiting decision under Needs you, with a back link to it", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    renderAsk();

    await screen.findByTestId(selectors.asks.question);
    expect(tab("asks")).toHaveAttribute("aria-current", "page");
    expect(tab("history")).not.toHaveAttribute("aria-current");
    expect(within(screen.getByRole("main")).getByRole("link", { name: en.layout.nav.asks })).toHaveAttribute("href", "/asks");
  });

  it("shows an answered decision, reached by a link, under History, with a back link to it", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    renderAsk();

    await screen.findByTestId(selectors.asks.question);
    expect(tab("history")).toHaveAttribute("aria-current", "page");
    expect(tab("asks")).not.toHaveAttribute("aria-current");
    expect(within(screen.getByRole("main")).getByRole("link", { name: en.layout.nav.history })).toHaveAttribute("href", "/history");
  });

  it("follows the list it was opened from, whatever state the decision is in", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    renderAsk("ask-1", "/history?q=goal");

    await screen.findByTestId(selectors.asks.question);
    expect(tab("history")).toHaveAttribute("aria-current", "page");
    expect(within(screen.getByRole("main")).getByRole("link", { name: en.layout.nav.history })).toHaveAttribute("href", "/history?q=goal");
  });

  it("stays under Needs you while the outcome of an answer just sent is read", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk)
      .mockResolvedValueOnce({ ask: makeAsk() } as never)
      .mockResolvedValue({ ask: makeAsk({ state: "answered", answer: "close", answerLabel: "Close the goal" }) } as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    renderAsk();

    await user.click(await optionByKey("close"));
    await user.click(send());
    await screen.findByText(en.packet.outcome.recorded.replace("{{option}}", "Close the goal"));
    expect(tab("asks")).toHaveAttribute("aria-current", "page");
  });
});

describe("how a closed decision reads", () => {
  it.each([
    ["answered", { answer: "close", answerLabel: "Close the goal" }],
    ["defaulted", { defaultAnswer: "close", defaultAppliedAt: "2026-11-14T08:00:00Z" }],
    ["expired", {}],
    ["withdrawn", { reason: "Replaced by a newer plan" }],
    ["dismissed", { reason: "not now" }],
  ])("leads with how it ended when %s, before the reasons and the record", async (state, over) => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ state, bigPicture: WHY, ...over }) } as never);
    renderAsk();

    const status = await screen.findByTestId(selectors.asks.status);
    const heading = screen.getByTestId(selectors.asks.question);
    const why = screen.getByText(WHY);
    expect(heading.compareDocumentPosition(status) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(status.compareDocumentPosition(why) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("keeps a waiting decision as it was: the default's countdown sits below the cover", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ bigPicture: WHY, defaultEligibleAt: new Date(Date.now() + 3 * 3_600_000 + 30_000).toISOString() }) } as never);
    renderAsk();

    const notice = await screen.findByTestId(selectors.asks.defaultNotice);
    expect(screen.getByText(WHY).compareDocumentPosition(notice) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});

describe("links in the decision text", () => {
  it("renders safe, phone-tappable web links throughout the prose", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        question: "Review [the run](https://example.test/run)",
        recommendationReason: "Details at [the report](https://example.test/report)",
        bigPicture: "Purpose: [overview](https://example.test/overview)",
        technicalDetails: "See [logs](https://example.test/logs) or https://example.test/raw-log",
      }),
    } as never);
    renderAsk();

    expect(await screen.findByRole("heading", { name: "Review the run" })).toBeInTheDocument();
    for (const [label, href] of [["the run", "https://example.test/run"], ["the report", "https://example.test/report"], ["overview", "https://example.test/overview"], ["logs", "https://example.test/logs"], ["https://example.test/raw-log", "https://example.test/raw-log"]]) {
      const link = screen.getByRole("link", { name: label });
      expect(link).toHaveAttribute("href", href);
      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
      expect(link).toHaveClass("break-words");
      expect(link).not.toHaveClass("break-all");
      // Links remain comfortable 44px touch targets even when they sit in a heading.
      expect(link).toHaveClass("min-h-11");
    }
  });

  it("shows a path or code span in the question as a monospace literal that can break anywhere", async () => {
    const path = "/home/op/Vrooli/.vrooli/work/delivery/captures/90-day-run/summary.json";
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ question: `Keep ${path} as the record?` }) } as never);
    renderAsk();

    const literal = (await screen.findByRole("heading", { level: 1 })).querySelector("code");
    expect(literal).toHaveTextContent(path);
    expect(literal).toHaveClass("font-mono", "[overflow-wrap:anywhere]", "text-base", "font-normal");
    expect(literal).toHaveAttribute("dir", "ltr");
  });

  it("opens local file references in the Hub viewer, including the context link", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({ question: "Review [the source](scenarios/notification-hub/README.md) and docs/internal/PROBLEMS.md.", contextUrl: "steward/RETURN.md" }),
    } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.asks.question)).toHaveTextContent("scenarios/notification-hub/README.md".slice(0, 0) + "Review the source");
    await user.click(screen.getByRole("button", { name: "the source" }));
    expect(await screen.findByTestId("viewer-stub")).toHaveTextContent("linked-artifact|README.md");
    expect(resolveLinkedArtifact).toHaveBeenCalledWith(ASK_ID, "scenarios/notification-hub/README.md");
  });

  it("returns from an unattached artifact to the same decision with its answer draft intact", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ technicalDetails: "Read /work/plan.md before answering." }) } as never);
    renderAsk();
    await user.click(await optionByKey("re-aim"));
    await user.click(await screen.findByRole("button", { name: "/work/plan.md" }));
    expect(await screen.findByTestId("viewer-stub")).toHaveTextContent("linked-artifact|plan.md");
    await user.click(screen.getByRole("button", { name: "Close viewer" }));
    expect(screen.queryByTestId("viewer-stub")).not.toBeInTheDocument();
    expect(await optionByKey("re-aim")).toBeChecked();
  });

  it("keeps the decision answerable when its linked file source is unavailable", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ technicalDetails: "Read /work/plan.md before answering." }) } as never);
    vi.mocked(resolveLinkedArtifact).mockRejectedValueOnce(new Error("source unavailable"));
    renderAsk();

    await user.click(await optionByKey("re-aim"));
    await user.click(await screen.findByRole("button", { name: "/work/plan.md" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(text.text.fileLinksUnavailable);
    expect(await optionByKey("re-aim")).toBeChecked();
    expect(send()).toBeEnabled();
    expect(screen.getByTestId(selectors.asks.retryFileLinks)).toBeEnabled();
    expect(screen.queryByTestId("viewer-stub")).not.toBeInTheDocument();
  });

  it("opens a path that names an attachment in the Hub's own viewer, without asking Web Console", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        question: "Check `/home/op/work/plan.md` first.",
        attachments: [makeAttachment({ name: "plan.md", sourcePath: "/home/op/work/plan.md" })],
      }),
    } as never);
    renderAsk();

    await user.click(await screen.findByRole("button", { name: "/home/op/work/plan.md" }));
    expect(screen.getByTestId("viewer-stub")).toHaveTextContent("plan.md|");
  });

  it("opens the context file in the viewer when it is attached", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ contextUrl: "plan.md", attachments: [makeAttachment({ name: "plan.md" })] }) } as never);
    renderAsk();

    await user.click(await screen.findByRole("button", { name: text.cover.more }));
    expect(screen.getByTestId("viewer-stub")).toHaveTextContent("plan.md|");
  });
});

const studioAsk = (overrides: Parameters<typeof makeAsk>[0] = {}) =>
  makeAsk({
    id: "ask-1",
    recommended: "set-b",
    options: [{ key: "set-b", label: "Set B" }, { key: "set-a", label: "Set A" }, { key: "other", label: "Other" }],
    attachments: [
      makeAttachment({ name: "brief.md" }),
      makeAttachment({ name: "set-a.png", previewKind: "image", mediaType: "image/png" }),
      makeAttachment({ name: "set-b.png", previewKind: "image", mediaType: "image/png" }),
      makeAttachment({ name: "shot-list.md", sourceStatus: "changed" }),
      makeAttachment({ name: "renders", folder: true, entries: [{ path: "walk.png", previewKind: "image", mediaType: "image/png" }] }),
    ],
    questions: [
      makeQuestion({
        id: "set",
        type: "variant",
        prompt: "Which studio set?",
        required: true,
        choices: [{ key: "a", label: "Warm desk", attachment: "set-a.png" }, { key: "b", label: "Open loft", attachment: "set-b.png" }],
        recommended: { selected: ["b"] },
        recommendationReason: "Reads better on a phone.",
      }),
      makeQuestion({ id: "shots", type: "multi_choice", prompt: "Which shots first?", choices: [{ key: "intro", label: "Intro" }, { key: "demo", label: "Demo" }], recommended: { selected: ["intro", "demo"] } }),
      makeQuestion({
        id: "brief",
        type: "section_review",
        prompt: "Review the brief",
        sections: [{ key: "goal", title: "Goal", attachment: "brief.md" }, { key: "timeline", title: "Timeline" }],
        recommended: { sections: [{ sectionKey: "timeline", verdict: "change", note: "two weeks" }] },
      }),
      makeQuestion({ id: "tone", type: "from-the-future", prompt: "Anything about tone?" }),
    ],
    ...overrides,
  });

describe("a decision packet", () => {
  beforeEach(() => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: studioAsk() } as never);
  });

  it("shows attachments as tiles, marks a changed file, and opens one in the viewer", async () => {
    const user = userEvent.setup();
    renderAsk();

    const tiles = await screen.findAllByTestId(selectors.packet.attachment);
    expect(tiles.map((tile) => tile.getAttribute("data-attachment-name"))).toEqual(["brief.md", "set-a.png", "set-b.png", "shot-list.md", "renders"]);
    expect(within(tiles[3]!).getByText(text.attachments.changed)).toBeInTheDocument();
    expect(tiles[1]!.querySelector("img")).toHaveAttribute("src", "/files/ask-1/set-a.png");
    await user.click(tiles[4]!);
    expect(screen.getByTestId("viewer-stub")).toHaveTextContent("renders|");
  });

  it("renders an optional observed block from the decision packet without a fidelity label", async () => {
    const block = {
      schemaVersion: "experience-packet-observed/v1",
      scenario: "audio-tools",
      generatedAt: "2026-11-24T13:07:33.901Z",
      build: "sha256:4e4e0059945491efb6f996b99828e12cac10c5ae0e24722b6dc9b6c4c4dc79f4",
      sessions: [{ id: "session-1", build: "sha256:4e4e0059945491efb6f996b99828e12cac10c5ae0e24722b6dc9b6c4c4dc79f4" }],
      window: { from: "2026-11-24T00:00:00.000Z", to: "2026-11-24T00:01:00.000Z", sessions: 1 },
      headline: "Repeated dictation flow observed",
      topFlow: { id: "flow-1", steps: ["dictation-record-start"], count: 4, medianDurationMs: 1000, totalDurationMs: 4000 },
      topFriction: { kind: "repeated_action", count: 2, routes: [], elements: [] },
      timeCostMs: 4000,
      reportDigest: `sha256:${"a".repeat(64)}`,
      caveat: "Local operator observation; advisory only, not a verdict.",
    };
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ attachments: [makeAttachment({ name: "packet-observed.json", mediaType: "application/json", previewKind: "text", sizeBytes: 900 })] }) } as never);
    vi.mocked(fetchAttachment).mockResolvedValue(new Response(JSON.stringify(block)));

    renderAsk();

    const observed = await screen.findByRole("note", { name: "Observed behavior packet context" });
    expect(within(observed).getByText("Repeated dictation flow observed")).toBeInTheDocument();
    expect(within(observed).getByText(/Top flow: dictation-record-start/)).toBeInTheDocument();
    expect(within(observed).getByText(/not a fidelity label/)).toBeInTheDocument();
    expect(within(observed).getAllByText(/sha256:4e4e0059945491ef/)).toHaveLength(2);
    expect(within(observed).getByText(/report digest:/)).toBeInTheDocument();
    expect(fetchAttachment).toHaveBeenCalledWith("ask-1", "packet-observed.json", "", expect.any(AbortSignal));
  });

  it("marks a malformed observed packet block not evaluated and leaves the decision available", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ attachments: [makeAttachment({ name: "packet-observed.json", mediaType: "application/json", sizeBytes: 32 })] }) } as never);
    vi.mocked(fetchAttachment).mockResolvedValue(new Response(JSON.stringify({ schemaVersion: "wrong" })));

    renderAsk();

    expect(await screen.findByRole("status", { name: "Observed behavior packet" })).toHaveTextContent("not evaluated");
    expect(screen.getByText("Close the goal")).toBeInTheDocument();
  });

  it("draws each question by type, with any unknown type as a text box", async () => {
    renderAsk();

    const questions = await screen.findAllByTestId(selectors.packet.question);
    expect(questions.map((question) => question.getAttribute("data-question-kind"))).toEqual(["variant", "multi_choice", "section_review", "text"]);
    expect(within(questions[0]!).getAllByRole("radio")).toHaveLength(2);
    expect(within(questions[1]!).getAllByRole("checkbox")).toHaveLength(2);
    expect(within(questions[2]!).getAllByRole("radiogroup")).toHaveLength(2);
    expect(within(questions[3]!).getByRole("textbox", { name: "Anything about tone?" })).toBeInTheDocument();
    expect(questions[0]!.querySelectorAll("img")).toHaveLength(2);
  });

  it("opens a variant's picture in the viewer without choosing it", async () => {
    const user = userEvent.setup();
    renderAsk();

    const question = (await screen.findAllByTestId(selectors.packet.question))[0]!;
    await user.click(within(question).getByRole("button", { name: text.form.viewChoice.replace("{{label}}", "Warm desk") }));
    expect(screen.getByTestId("viewer-stub")).toHaveTextContent("set-a.png|");
    expect(within(question).getAllByRole("radio").every((radio) => !(radio as HTMLInputElement).checked)).toBe(true);
  });

  it("binds a required question to the requester's options, and never to Other", async () => {
    const user = userEvent.setup();
    renderAsk();

    await user.click(await optionByKey("set-a"));
    expect(send()).toBeDisabled();
    expect(screen.getByText(text.form.blockedRequired.replace("{{prompt}}", "Which studio set?"))).toBeInTheDocument();

    await user.click(await optionByKey("other"));
    await user.type(screen.getByLabelText(text.form.otherPrompt), "Neither works");
    expect(send()).toBeEnabled();
  });

  it("pre-fills everything recommended without sending it", async () => {
    const user = userEvent.setup();
    renderAsk();

    await user.click(await screen.findByTestId(selectors.packet.acceptRecommended));

    expect(await optionByKey("set-b")).toBeChecked();
    const questions = screen.getAllByTestId(selectors.packet.question);
    expect(within(questions[0]!).getByRole("radio", { name: /Open loft/ })).toBeChecked();
    expect(within(questions[1]!).getAllByRole("checkbox").every((box) => (box as HTMLInputElement).checked)).toBe(true);
    expect(within(questions[2]!).getByRole("radio", { name: text.form.verdicts.change, checked: true })).toBeInTheDocument();
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(send()).toBeEnabled();
  });

  it("sends the main answer, typed responses and notes pinned to files in one answer", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    writeAskDraft("owner-1", "ask-1", {
      ...emptyDraft(),
      answerChanges: { pace: { label: "Use Wednesday", value: "wednesday" } },
      answerImages: [{ name: "pace.png", sha256: "a".repeat(64), mediaType: "image/png", sizeBytes: 4 }],
    });
    renderAsk();

    await user.click(await screen.findByTestId(selectors.packet.acceptRecommended));
    const tone = within(screen.getAllByTestId(selectors.packet.question)[3]!).getByRole("textbox");
    await user.type(tone, "Warmer");
    await user.click(send());

    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    expect(conversationsClient.answer).toHaveBeenCalledWith({
      askId: "ask-1",
      answer: "set-b",
      note: "",
      responses: [
        { questionId: "set", selected: ["b"], text: "", sections: [] },
        { questionId: "shots", selected: ["intro", "demo"], text: "", sections: [] },
        { questionId: "brief", selected: [], text: "", sections: [{ sectionKey: "goal", verdict: "approve", note: "" }, { sectionKey: "timeline", verdict: "change", note: "two weeks" }] },
        { questionId: "tone", selected: [], text: "Warmer", sections: [] },
      ],
      attachmentNotes: [],
      changes: { pace: "wednesday" },
      images: [{ name: "pace.png", sha256: "a".repeat(64), mediaType: "image/png", sizeBytes: 4n }],
    });
  });

  it("lists the notes pinned to files for editing, and sends them with the answer", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    // The viewer pins a note into the unsent draft.
    writeAskDraft("owner-1", "ask-1", withAttachmentNote(emptyDraft(), "brief.md", "", "Too long"));
    renderAsk();

    const note = await screen.findByRole("textbox", { name: text.form.noteOn.replace("{{name}}", "brief.md") });
    expect(note).toHaveValue("Too long");
    await user.type(note, "!");
    await user.click(await screen.findByTestId(selectors.packet.acceptRecommended));
    await user.click(send());

    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
    expect(vi.mocked(conversationsClient.answer).mock.calls[0]?.[0]).toMatchObject({ attachmentNotes: [{ attachment: "brief.md", path: "", note: "Too long!" }] });
  });

  it("is accessible", async () => {
    const view = renderAsk();
    await screen.findAllByTestId(selectors.packet.question);
    await expectNoA11yViolations(view.container);
  });

  describe("on a phone with several questions", () => {
    beforeEach(() => setPhone(true));

    it("keeps the packet visible and opens all questions in the choice drawer", async () => {
      const user = userEvent.setup();
      vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
      renderAsk();

      expect(await screen.findAllByTestId(selectors.packet.attachment)).toHaveLength(5);
      expect(screen.queryByTestId(selectors.packet.form)).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: en.packet.form.choose }));
      expect(screen.getByRole("dialog", { name: en.packet.form.stepDecision })).toBeInTheDocument();
      expect(screen.getAllByTestId(selectors.packet.question)).toHaveLength(4);
      await user.click(await optionByKey("set-b"));
      await user.click(screen.getByRole("radio", { name: /Open loft/ }));
      await user.click(screen.getByRole("checkbox", { name: /Intro/ }));
      await user.type(screen.getByRole("textbox", { name: "Anything about tone?" }), "Warmer");
      await user.click(within(screen.getByRole("dialog", { name: en.packet.form.stepDecision })).getByRole("button", { name: en.packet.form.closeChoices }));
      expect(screen.getByRole("button", { name: TEXT_SET_B })).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Send" }));
      await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledTimes(1));
      expect(vi.mocked(conversationsClient.answer).mock.calls[0]?.[0]).toMatchObject({ answer: "set-b" });
    });

    it("keeps sending disabled when recommended choices leave a required question unanswered", async () => {
      const user = userEvent.setup();
      vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: studioAsk({ questions: studioAsk().questions.map((question) => (question.id === "tone" ? makeQuestion({ id: "tone", type: "text", prompt: "Tone?", required: true }) : question)) }) } as never);
      renderAsk();

      await user.click(await screen.findByRole("button", { name: en.packet.form.choose }));
      await user.click(screen.getByTestId(selectors.packet.acceptRecommended));
      expect(screen.getAllByTestId(selectors.packet.question)).toHaveLength(4);
      await user.click(within(screen.getByRole("dialog", { name: en.packet.form.stepDecision })).getByRole("button", { name: en.packet.form.closeChoices }));
      expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    });
  });
});

describe("asking for more context", () => {
  it("sends the question, keeps the decision open, and says it is waiting for the agent", async () => {
    const user = userEvent.setup();
    const waiting = makeAsk({
      openContextRequestCount: 1,
      contextRequests: [create(AskContextRequestSchema, { id: "c1", question: "What does the gate mean?", requestedAt: "2026-11-13T10:00:00Z" })],
    });
    vi.mocked(conversationsClient.getAsk).mockResolvedValueOnce({ ask: makeAsk() } as never).mockResolvedValue({ ask: waiting } as never);
    vi.mocked(conversationsClient.requestContext).mockResolvedValue({} as never);
    renderAsk();

    await user.click(await screen.findByTestId(selectors.packet.needContext));
    const sheet = screen.getByRole("dialog");
    expect(within(sheet).getByTestId(selectors.packet.sendContext)).toBeDisabled();
    await user.type(within(sheet).getByTestId(selectors.packet.contextQuestion), "  What does the gate mean?  ");
    await user.click(within(sheet).getByTestId(selectors.packet.sendContext));

    await waitFor(() => expect(conversationsClient.requestContext).toHaveBeenCalledWith({ askId: "ask-1", question: "What does the gate mean?" }));
    expect(await screen.findByText(text.context.waiting)).toBeInTheDocument();
    expect(screen.getByText(TEXT_WHAT_DOES_THE_GATE)).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(await screen.findAllByTestId(selectors.asks.option)).toHaveLength(3);
  });

  it("shows a plain line instead of a dead button when the sender does not answer questions", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ canRequestContext: false }) } as never);
    renderAsk();

    expect(await screen.findByTestId(selectors.packet.contextUnavailable)).toHaveTextContent(text.context.unavailable);
    expect(screen.queryByTestId(selectors.packet.needContext)).not.toBeInTheDocument();
  });

  it("closes the sheet and drops the button when the Hub says the sender stopped accepting questions", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValueOnce({ ask: makeAsk() } as never).mockResolvedValue({ ask: makeAsk({ canRequestContext: false }) } as never);
    vi.mocked(conversationsClient.requestContext).mockRejectedValue(new ConnectError("context_not_accepted", Code.FailedPrecondition, new Headers({ "Hub-Error-Reason": "context_not_accepted" })));
    renderAsk();

    await user.click(await screen.findByTestId(selectors.packet.needContext));
    await user.type(screen.getByTestId(selectors.packet.contextQuestion), "What is the gate?");
    await user.click(screen.getByTestId(selectors.packet.sendContext));

    expect(await screen.findByTestId(selectors.packet.contextUnavailable)).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.needContext)).not.toBeInTheDocument();
  });

  it("keeps the typed question and says so when sending fails", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk() } as never);
    vi.mocked(conversationsClient.requestContext).mockRejectedValue(new Error("offline"));
    renderAsk();

    await user.click(await screen.findByTestId(selectors.packet.needContext));
    await user.type(screen.getByTestId(selectors.packet.contextQuestion), "What is the gate?");
    await user.click(screen.getByTestId(selectors.packet.sendContext));

    expect(await screen.findByText(text.context.error)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.packet.contextQuestion)).toHaveValue("What is the gate?");
    expect(screen.getByTestId(selectors.packet.sendContext)).toBeEnabled();
  });

  it("says the decision was updated after the question, and shows what changed", async () => {
    const options = makeAsk().options.map((option) => create(AskOptionSchema, option));
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: makeAsk({
        question: "Does the gate (the review step) pass?",
        revisionCount: 1,
        revisions: [create(AskRevisionSchema, { revision: 1, question: "Does the gate pass?", options, revisedAt: "2026-11-13T10:00:00Z" })],
        contextRequests: [create(AskContextRequestSchema, { id: "c1", question: "What gate?", requestedAt: "2026-11-13T09:00:00Z", addressedAt: "2026-11-13T10:00:00Z", addressedRevision: 1 })],
      }),
    } as never);
    const user = userEvent.setup();
    renderAsk();

    expect(await screen.findByText(text.context.updated)).toBeInTheDocument();
    expect(screen.queryByText(text.context.waiting)).not.toBeInTheDocument();
    const revisions = document.getElementById("packet-revisions") as HTMLDetailsElement;
    expect(revisions.open).toBe(false);
    await user.click(screen.getByRole("button", { name: text.context.seeChanges }));
    expect(revisions.open).toBe(true);
    expect(within(revisions).getByText(TEXT_DOES_THE_GATE_PASS)).toBeInTheDocument();
  });

  describe("the requester's answer to a question", () => {
    const request = (id: string, question: string, addressedAt: string, response = "") =>
      create(AskContextRequestSchema, { id, question, requestedAt: "2026-11-13T09:00:00Z", addressedAt, addressedRevision: 0, response });
    const open = (contextRequests: ReturnType<typeof request>[]) => {
      vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ contextRequests }) } as never);
      renderAsk();
      return screen.findByTestId(selectors.packet.contextRequests);
    };

    it("shows the question and the answer in full, line breaks kept, with no changes to offer when nothing was revised", async () => {
      const block = await open([request("c1", GATE_QUESTION, "2026-11-13T10:00:00Z", "The review step.\nIt runs before merge.")]);

      expect(within(block).getByText(text.context.yourQuestion)).toBeInTheDocument();
      expect(within(block).getByText(GATE_QUESTION)).toBeInTheDocument();
      const answer = within(block).getByText(text.context.answer).nextElementSibling as HTMLElement;
      expect(answer).toHaveTextContent(REVIEW_STEP);
      expect(answer.textContent).toBe("The review step.\nIt runs before merge.");
      expect(answer).not.toHaveClass("line-clamp-4");
      expect(within(block).queryByRole("button", { name: text.context.seeChanges })).not.toBeInTheDocument();
    });

    it("draws an answered question without a written answer as before: the question, no Answer row", async () => {
      const block = await open([request("c1", GATE_QUESTION, "2026-11-13T10:00:00Z")]);

      expect(within(block).getByText(GATE_QUESTION)).toBeInTheDocument();
      expect(within(block).queryByText(text.context.answer)).not.toBeInTheDocument();
      expect(within(block).queryByText(text.context.yourQuestion)).not.toBeInTheDocument();
    });

    it("says the answer once when one reply covered two questions", async () => {
      const block = await open([
        request("c1", GATE_QUESTION, "2026-11-13T10:00:00Z", SHARED_ANSWER),
        request("c2", BUILD_QUESTION, "2026-11-13T10:00:00Z", SHARED_ANSWER),
      ]);

      expect(within(block).getAllByText(SHARED_ANSWER)).toHaveLength(1);
      expect(within(block).getByText(GATE_QUESTION)).toBeInTheDocument();
      expect(within(block).getByText(BUILD_QUESTION)).toBeInTheDocument();
    });

    it("keeps the latest reply in view and folds earlier rounds away, answers inside them, even when no round changed a term", async () => {
      const block = await open([
        request("c1", "First question", "2026-11-13T10:00:00Z", "First answer"),
        request("c2", "Second question", "2026-11-13T11:00:00Z", LATEST_ANSWER),
      ]);

      const folded = block.querySelector("details") as HTMLElement;
      expect(folded).not.toContainElement(within(block).getByText(LATEST_ANSWER));
      expect(folded).toHaveTextContent("First question");
      expect(folded).toHaveTextContent("First answer");
      expect(folded).not.toHaveTextContent(LATEST_ANSWER);
      expect(within(folded).getByText(text.context.earlier.replace("{{count}}", "1"))).toBeInTheDocument();
    });

    it("treats rows carried over from a replaced decision as earlier, not as the latest reply", async () => {
      const block = await open([
        request("c1", "Carried question", "2026-11-12T10:00:00Z", CARRIED_ANSWER),
        request("c2", "Fresh question", "2026-11-13T10:00:00Z", FRESH_ANSWER),
      ]);

      expect(block.querySelector("details")).toHaveTextContent(CARRIED_ANSWER);
      expect(within(block).getByText(FRESH_ANSWER)).toBeInTheDocument();
    });
  });
});
