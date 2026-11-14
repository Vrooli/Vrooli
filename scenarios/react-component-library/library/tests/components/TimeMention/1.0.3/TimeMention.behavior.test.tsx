import { act, cleanup, fireEvent, screen, within } from "@testing-library/react";
import ReactMarkdown from "react-markdown";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../../../ui/src/test-utils";
import {
  MarkdownTimeMention,
  TIME_MENTION_DISPLAY_STORAGE_KEY,
  TimeMention,
  TimeMentionProvider,
  detectTimeMentions,
  remarkTimeMentions,
  resolveTimeMention,
  setStoredTimeMentionDisplay,
  type TimeFrame,
} from "@vrooli/react-component-library/TimeMention/1.0.3";

// The scene from the operator's screenshot: a message sent at 9:27 PM New York
// time on Oct 28, 2026 promising an 08:53:33 UTC cutoff, read four minutes later.
const SENT = Date.UTC(2026, 9, 29, 1, 27);
const NOW = SENT + 4 * 60_000;
const ZONE = "America/New_York";
const frame = (overrides: Partial<TimeFrame> = {}): TimeFrame => ({ anchor: SENT, now: NOW, timeZone: ZONE, locale: "en-US", ...overrides });

const kinds = (text: string) => detectTimeMentions(text).map((m) => [m.text, m.data.kind]);
const squash = (value: string | null | undefined) => (value ?? "").replace(/\s+/g, " ");

beforeEach(() => {
  localStorage.removeItem(TIME_MENTION_DISPLAY_STORAGE_KEY);
  setStoredTimeMentionDisplay("annotate");
});
afterEach(cleanup);

describe("time mention grammar", () => {
  it("finds a zoned clock time and reads the nearby word cutoff as a deadline", () => {
    const [mention] = detectTimeMentions("keep the original 08:53:33 UTC trial cutoff, and continue");
    expect(mention).toMatchObject({ text: "08:53:33 UTC", role: "deadline", data: { kind: "clock", hour: 8, minute: 53, second: 33, offset: 0 } });
  });

  it("reads ISO 8601 with a zone as an exact moment and without one as unzoned", () => {
    const [zoned] = detectTimeMentions("expires at 2026-10-29T03:02:00Z.");
    expect(zoned?.data).toEqual({ kind: "instant", source: "iso", at: Date.UTC(2026, 9, 29, 3, 2), seconds: true, offset: 0 });
    expect(zoned?.role).toBe("deadline");
    expect(detectTimeMentions("logged 2026-10-29 08:53")[0]?.data.kind).toBe("unzoned");
    expect(detectTimeMentions("at 2026-10-29T08:53:00-04:00")[0]?.data).toMatchObject({ at: Date.UTC(2026, 9, 29, 12, 53) });
  });

  it("decodes Unix timestamps only beside a word that names a time", () => {
    expect(kinds("lease_ts 1793228013")).toEqual([["1793228013", "instant"]]);
    expect(kinds("created_at=1793228013123")).toEqual([["1793228013123", "instant"]]);
    expect(kinds("row id 1793228013 and 1793228013 bytes")).toEqual([]);
  });

  it("recognises written dates with weekdays, but leaves lowercase prose alone", () => {
    expect(detectTimeMentions("baseline (Wednesday, Oct 29) kept")[0]?.data).toEqual({ kind: "date", year: null, month: 10, day: 29, weekday: 3 });
    expect(detectTimeMentions("on March 3, 2026")[0]?.data).toMatchObject({ year: 2026, month: 3, day: 3 });
    expect(kinds("we may 3 times retry")).toEqual([]);
    expect(kinds("Feb 30 is not a day")).toEqual([]);
  });

  it("never matches versions, ratios, paths or file names", () => {
    expect(kinds("bump to 1.10.30, keep 16:9 and 3:2, open notes/2026-10-29.md or 2026-10-29-report.md")).toEqual([]);
  });

  it("takes a deadline word from surrounding context outside the text", () => {
    expect(detectTimeMentions("08:53:33 UTC", { before: "Keep the original ", after: " trial cutoff." })[0]?.role).toBe("deadline");
    expect(detectTimeMentions("08:53:33 UTC", { before: "Logged at ", after: " by the runner." })[0]?.role).toBeNull();
  });

  it("reads relative words and promises", () => {
    expect(kinds("Yesterday's baseline; I'll check back in 20 minutes; started 3 hours ago")).toEqual([
      ["Yesterday", "relative-day"],
      ["in 20 minutes", "relative-offset"],
      ["3 hours ago", "relative-offset"],
    ]);
    expect(detectTimeMentions("check back in an hour")[0]).toMatchObject({ role: "promise", data: { ms: 3_600_000 } });
  });

  it("only offers durations whose readable form is shorter", () => {
    expect(kinds("ran 7380s on a PT90M budget, 45s idle, 350ms p50, the 1990s")).toEqual([
      ["7380s", "duration"],
      ["PT90M", "duration"],
    ]);
  });

  it("takes a bare clock time only after a preposition, starting at the digits", () => {
    const [mention] = detectTimeMentions("the planner sync is at 9:30 tomorrow");
    expect(mention).toMatchObject({ text: "9:30", data: { kind: "unzoned", hour: 9, minute: 30 } });
    expect(kinds("score was 9:30")).toEqual([]);
  });
});

describe("resolving into the reader's frame", () => {
  it("dates a zoned clock time by the occurrence nearest to when it was sent", () => {
    const [mention] = detectTimeMentions("08:53:33 UTC cutoff");
    const resolved = resolveTimeMention(mention!.data, mention!.role, frame());
    expect(resolved).toMatchObject({ instant: Date.UTC(2026, 9, 29, 8, 53, 33), dateKnown: true, role: "deadline" });
    expect(resolved?.notes.map((n) => n.key)).toContain("data-display.time-mention.note.date-inferred");
  });

  it("warns about the other day only when both readings are plausible", () => {
    const warns = (text: string) =>
      (resolveTimeMention(detectTimeMentions(text)[0]!.data, null, frame())?.notes ?? []).some((n) => n.key.endsWith("date-uncertain"));
    // 7.4h after sending vs 16.6h before: clearly the coming one.
    expect(warns("08:53:33 UTC")).toBe(false);
    // 11.5h after vs 12.5h before: either could be meant.
    expect(warns("13:00 UTC")).toBe(true);
  });

  it("without a send time converts only the clock and drops the countdown", () => {
    const [mention] = detectTimeMentions("08:53:33 UTC cutoff");
    const resolved = resolveTimeMention(mention!.data, mention!.role, frame({ anchor: null }));
    expect(resolved).toMatchObject({ dateKnown: false, role: null });
  });

  it("hides words that need a send time when there is none, and while they are still true", () => {
    const [yesterday, later] = detectTimeMentions("yesterday, and again in 20 minutes");
    expect(resolveTimeMention(yesterday!.data, null, frame({ anchor: null }))).toBeNull();
    expect(resolveTimeMention(later!.data, later!.role, frame({ anchor: null }))).toBeNull();
    expect(resolveTimeMention(yesterday!.data, null, frame())).toBeNull();
    const nextDay = resolveTimeMention(yesterday!.data, null, frame({ now: SENT + 30 * 3_600_000 }));
    expect(nextDay?.day).toBe(Date.UTC(2026, 9, 27, 12));
  });

  it("flags a weekday that does not match its date", () => {
    const [mention] = detectTimeMentions("Wednesday, Oct 29");
    const resolved = resolveTimeMention(mention!.data, null, frame());
    expect(resolved?.lint).toMatchObject({ values: { weekday: "Thursday" } });
  });

  it("names the other readings of an ambiguous zone", () => {
    const [mention] = detectTimeMentions("sweep at 02:00 CST");
    const notes = resolveTimeMention(mention!.data, null, frame())?.notes ?? [];
    expect(notes.find((n) => n.key.endsWith("ambiguous-zone"))?.values?.others).toContain("China");
  });
});

function renderMention(text: string, props: { anchor?: number | null } = {}) {
  const [mention] = detectTimeMentions(text);
  if (!mention) throw new Error(`no mention in ${text}`);
  return renderWithProviders(
    <TimeMentionProvider anchor={props.anchor === undefined ? SENT : props.anchor} now={NOW} timeZone={ZONE} locale="en-US">
      <p>
        <TimeMention text={mention.text} data={mention.data} role={mention.role} />
      </p>
    </TimeMentionProvider>,
  );
}

describe("TimeMention", () => {
  it("keeps the written text and adds the reader's time and a countdown beside it", () => {
    const { container } = renderMention("the 08:53:33 UTC cutoff");
    const mention = container.querySelector("[data-rcl-time-mention]");
    expect(mention?.querySelector("time")?.textContent).toBe("08:53:33 UTC");
    expect(squash(mention?.querySelector("[data-rcl-time-mention-tag]")?.textContent)).toBe("4:53 AM tomorrow");
    expect(mention?.querySelector("[data-rcl-time-mention-badge]")?.textContent).toBe("in 7h 22m");
    expect(squash(mention?.getAttribute("aria-label"))).toBe("08:53:33 UTC, 4:53 AM tomorrow, in 7h 22m");
    // The anchor wrapper is phrasing content, so the paragraph stays valid.
    expect(container.querySelector("p div")).toBeNull();
  });

  it("adds nothing when the writer's zone is already the reader's", () => {
    const { container } = renderMention("at 9:00 PM EDT");
    expect(container.querySelector("[data-rcl-time-mention-tag]")).toBeNull();
  });

  it("marks a night-time moment for the reader", () => {
    const { container } = renderMention("sweep at 02:00 UTC");
    expect(container.querySelector("[data-rcl-time-mention-tag] [data-night]")).not.toBeNull();
  });

  it("renders words that need a send time as plain text when there is none", () => {
    const { container } = renderMention("check back in 20 minutes", { anchor: null });
    expect(container.querySelector("[data-rcl-time-mention]")).toBeNull();
    expect(container.textContent).toBe("in 20 minutes");
  });

  it("opens details and switches every mention's display from the menu", async () => {
    const text = "cutoff 08:53:33 UTC and lease expires at 2026-10-29T03:02:00Z";
    const mentions = detectTimeMentions(text);
    const { container } = renderWithProviders(
      <TimeMentionProvider anchor={SENT} now={NOW} timeZone={ZONE} locale="en-US">
        <p>
          {mentions.map((m) => (
            <TimeMention key={m.start} text={m.text} data={m.data} role={m.role} />
          ))}
        </p>
      </TimeMentionProvider>,
    );
    fireEvent.click(container.querySelector("[data-rcl-time-mention]")!);
    const card = await screen.findByRole("dialog", { name: "Time details" });
    expect(squash(within(card).getByText(/4:53:33/).textContent)).toBe("4:53:33 AM");
    expect(within(card).getByText("2026-10-29T08:53:33Z")).toBeTruthy();
    expect(within(card).getByText("7h 26m after")).toBeTruthy();

    act(() => {
      fireEvent.click(within(card).getByRole("button", { name: "Relative" }));
    });
    const faces = [...container.querySelectorAll("[data-rcl-time-mention] time")].map((el) => el.textContent);
    expect(faces).toEqual(["in 7h 22m", "in 1h 31m"]);
    expect(localStorage.getItem(TIME_MENTION_DISPLAY_STORAGE_KEY)).toBe("relative");
  });

  it("lets a host own the display choice instead of the shared preference", () => {
    const [mention] = detectTimeMentions("the 08:53:33 UTC cutoff");
    const { container } = renderWithProviders(
      <TimeMentionProvider anchor={SENT} now={NOW} timeZone={ZONE} locale="en-US" display="local">
        <TimeMention text={mention!.text} data={mention!.data} role={mention!.role} />
      </TimeMentionProvider>,
    );
    expect(squash(container.querySelector("[data-rcl-time-mention] time")?.textContent)).toBe("4:53 AM tomorrow");
    expect(container.querySelector("[data-rcl-time-mention-tag]")).toBeNull();
  });

  it("opens from the keyboard", async () => {
    const { container } = renderMention("the 08:53:33 UTC cutoff");
    fireEvent.keyDown(container.querySelector("[data-rcl-time-mention]")!, { key: "Enter" });
    expect(await screen.findByRole("dialog", { name: "Time details" })).toBeTruthy();
  });
});

describe("remarkTimeMentions", () => {
  it("turns prose mentions into interactive times and leaves code and links literal", () => {
    const { container } = renderWithProviders(
      <TimeMentionProvider anchor={SENT} now={NOW} timeZone={ZONE} locale="en-US">
        <ReactMarkdown remarkPlugins={[remarkTimeMentions]} components={{ time: MarkdownTimeMention }}>
          {"Keep the **08:53:33 UTC** cutoff. Config `retry_at=08:53:33Z` and [run at 09:00 UTC](https://example.com) stay."}
        </ReactMarkdown>
      </TimeMentionProvider>,
    );
    const mentions = container.querySelectorAll("[data-rcl-time-mention]");
    expect(mentions).toHaveLength(1);
    expect(mentions[0]?.closest("strong")).not.toBeNull();
    expect(container.querySelector("code")?.textContent).toBe("retry_at=08:53:33Z");
    expect(container.querySelector("a")?.textContent).toBe("run at 09:00 UTC");
  });

  it("reads a deadline from the rest of the paragraph when the time is emphasized", () => {
    const { container } = renderWithProviders(
      <TimeMentionProvider anchor={SENT} now={NOW} timeZone={ZONE} locale="en-US">
        <ReactMarkdown remarkPlugins={[remarkTimeMentions]} components={{ time: MarkdownTimeMention }}>
          {"Keep the original **08:53:33 UTC** trial cutoff. Sweep at **02:00 UTC** nightly."}
        </ReactMarkdown>
      </TimeMentionProvider>,
    );
    const badges = [...container.querySelectorAll("[data-rcl-time-mention]")].map((m) => m.querySelector("[data-rcl-time-mention-badge]") !== null);
    expect(badges).toEqual([true, false]);
  });
});
