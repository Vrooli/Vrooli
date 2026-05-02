import { describe, expect, it } from "vitest";
import {
  applyPhaseAction,
  buildPhaseEnvelope,
  isPhaseQuickActionKey,
  prunePhaseActions,
  type PhaseQuickActionKey,
} from "./phase-composer-envelope";

describe("buildPhaseEnvelope", () => {
  it("emits empty blocks when no actions or items are selected", () => {
    const xml = buildPhaseEnvelope({ phase: "investigate", items: [], actions: [], note: "" });
    expect(xml).toContain('<phase name="investigate" />');
    expect(xml).toContain("<selection></selection>");
    expect(xml).toContain("<requested_actions></requested_actions>");
    expect(xml).toContain("<user_note>");
  });

  it("renders the user note even when actions and items are empty", () => {
    const xml = buildPhaseEnvelope({
      phase: "plan",
      items: [],
      actions: [],
      note: "Reconsider the approach.",
    });
    expect(xml).toContain("Reconsider the approach.");
  });

  it("renders selected actions and items as XML", () => {
    const xml = buildPhaseEnvelope({
      phase: "execute",
      items: ["execute/do-thing", "fix/typo"],
      actions: ["focus_on_items", "tighten_scope"],
      note: "",
    });
    expect(xml).toContain('<item ref="execute/do-thing" />');
    expect(xml).toContain('<item ref="fix/typo" />');
    expect(xml).toContain('<action name="focus_on_items" />');
    expect(xml).toContain('<action name="tighten_scope" />');
  });

  it("escapes XML-special characters in phase name, items, and actions", () => {
    const xml = buildPhaseEnvelope({
      phase: 'rude<phase>"',
      items: ['kind/<bad&"'],
      actions: ["continue_from_prior"],
      note: "raw <chars> & \"quoted\" should NOT be escaped in note",
    });
    expect(xml).toContain('<phase name="rude&lt;phase&gt;&quot;" />');
    expect(xml).toContain('<item ref="kind/&lt;bad&amp;&quot;" />');
    // Note body is intentionally left raw — operators may want to paste markdown/code.
    expect(xml).toContain('raw <chars> & "quoted" should NOT be escaped in note');
  });

  it("trims whitespace around the user note", () => {
    const xml = buildPhaseEnvelope({
      phase: "review",
      items: [],
      actions: [],
      note: "   trim me   \n\n",
    });
    expect(xml).toContain("trim me");
    expect(xml).not.toContain("   trim me   ");
  });

  it("composes a full envelope deterministically", () => {
    const xml = buildPhaseEnvelope({
      phase: "investigate",
      items: ["execute/do-thing"],
      actions: ["focus_on_items"],
      note: "Look here first.",
    });
    expect(xml).toBe(
      [
        "<phase_request>",
        '  <phase name="investigate" />',
        "  <selection>",
        '    <item ref="execute/do-thing" />',
        "  </selection>",
        "  <requested_actions>",
        '    <action name="focus_on_items" />',
        "  </requested_actions>",
        "  <user_note>",
        "Look here first.",
        "  </user_note>",
        "</phase_request>",
      ].join("\n"),
    );
  });
});

describe("applyPhaseAction", () => {
  it("toggles an action off when already selected", () => {
    const prev = new Set<PhaseQuickActionKey>(["continue_from_prior"]);
    expect(applyPhaseAction(prev, "continue_from_prior")).toEqual(new Set());
  });

  it("makes tighten_scope and expand_scope mutually exclusive", () => {
    let s = new Set<PhaseQuickActionKey>();
    s = applyPhaseAction(s, "tighten_scope");
    expect(s).toEqual(new Set(["tighten_scope"]));
    s = applyPhaseAction(s, "expand_scope");
    expect(s).toEqual(new Set(["expand_scope"]));
  });

  it("makes continue_from_prior and reset_and_reinvestigate mutually exclusive", () => {
    let s = new Set<PhaseQuickActionKey>();
    s = applyPhaseAction(s, "continue_from_prior");
    s = applyPhaseAction(s, "reset_and_reinvestigate");
    expect(s).toEqual(new Set(["reset_and_reinvestigate"]));
  });

  it("stacks non-exclusive actions", () => {
    let s = new Set<PhaseQuickActionKey>();
    s = applyPhaseAction(s, "continue_from_prior");
    s = applyPhaseAction(s, "focus_on_items");
    s = applyPhaseAction(s, "skip_unblock");
    expect(s).toEqual(
      new Set(["continue_from_prior", "focus_on_items", "skip_unblock"]),
    );
  });
});

describe("prunePhaseActions", () => {
  it("drops focus_on_items when item selection is empty", () => {
    const prev = new Set<PhaseQuickActionKey>(["focus_on_items", "tighten_scope"]);
    const next = prunePhaseActions(prev, { itemSelectionSize: 0, phaseStartable: false });
    expect(next.has("focus_on_items")).toBe(false);
    expect(next.has("tighten_scope")).toBe(true);
  });

  it("drops skip_unblock when the phase is now startable", () => {
    const prev = new Set<PhaseQuickActionKey>(["skip_unblock", "continue_from_prior"]);
    const next = prunePhaseActions(prev, { itemSelectionSize: 0, phaseStartable: true });
    expect(next.has("skip_unblock")).toBe(false);
    expect(next.has("continue_from_prior")).toBe(true);
  });
});

describe("isPhaseQuickActionKey", () => {
  it("recognizes valid keys", () => {
    expect(isPhaseQuickActionKey("focus_on_items")).toBe(true);
  });
  it("rejects unknown keys", () => {
    expect(isPhaseQuickActionKey("teleport")).toBe(false);
  });
});
