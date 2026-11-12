/**
 * The exact requests behind every outward action, observed at the wire.
 * Nothing here reaches a real desk: `stubWire` answers every call.
 */
import { describe, expect, it } from "vitest";

import { stubWire } from "../test-utils";
import { ChangeSubject } from "./activity";
import { ControlAction, launchApi } from "./launch";
import { EntryKind, operatorApi } from "./operator";

const OPERATOR = "/vrooli.research_desk.v1.operator.OperatorService";
const LAUNCH = "/vrooli.research_desk.v1.launch.LaunchService";

describe("operator requests", () => {
  it("answers an item with an option and an optional note", async () => {
    const calls = stubWire();
    await operatorApi.answer("programme-five-20261112", "AV4", "keep", "tell me if AV6 needs it");
    expect(calls).toEqual([
      { path: `${OPERATOR}/Answer`, body: { commissionId: "programme-five-20261112", itemId: "AV4", optionKey: "keep", note: "tell me if AV6 needs it" } },
    ]);
  });

  it("writes direction and wind-down entries in the operator's words", async () => {
    const calls = stubWire();
    await operatorApi.addEntry("c1", EntryKind.DIRECTION, "Focus on IT6.");
    await operatorApi.addEntry("c1", EntryKind.WIND_DOWN, "Finish the unit, then close out.");
    expect(calls.map((c) => c.body)).toEqual([
      { commissionId: "c1", kind: "ENTRY_KIND_DIRECTION", text: "Focus on IT6." },
      { commissionId: "c1", kind: "ENTRY_KIND_WIND_DOWN", text: "Finish the unit, then close out." },
    ]);
  });

  it("records a look as an operator write, never as part of reading changes, and one look at everything for Caught up", async () => {
    const calls = stubWire();
    await operatorApi.markSeen(ChangeSubject.COMMISSION, "c1");
    await operatorApi.markAllSeen();
    expect(calls).toEqual([
      { path: `${OPERATOR}/MarkSeen`, body: { subject: "CHANGE_SUBJECT_COMMISSION", id: "c1" } },
      { path: `${OPERATOR}/MarkSeen`, body: { all: true } },
    ]);
  });

  it("asks each write for its exact text as a dry run, which the API never writes (rulings P5)", async () => {
    const calls = stubWire();
    await operatorApi.answer("c1", "AV4", "keep", "", true);
    await operatorApi.addEntry("c1", EntryKind.DIRECTION, "Focus on IT6.", true);
    await operatorApi.addVerdict("r1", "debt-reduction", "Keep going.", true);
    expect(calls.map((c) => c.body)).toEqual([
      { commissionId: "c1", itemId: "AV4", optionKey: "keep", dryRun: true },
      { commissionId: "c1", kind: "ENTRY_KIND_DIRECTION", text: "Focus on IT6.", dryRun: true },
      { reviewId: "r1", topic: "debt-reduction", text: "Keep going.", dryRun: true },
    ]);
  });

  it("flips one commission's courier switch, or the global one", async () => {
    const calls = stubWire();
    await operatorApi.setCourier({ commissionId: "c1" }, true);
    await operatorApi.setCourier({ global: true }, false);
    expect(calls).toEqual([
      { path: `${OPERATOR}/SetCourier`, body: { commissionId: "c1", on: true } },
      { path: `${OPERATOR}/SetCourier`, body: { global: true } },
    ]);
  });

  it("enables steering and records a verdict by id", async () => {
    const calls = stubWire();
    await operatorApi.enableSteering("c1");
    await operatorApi.addVerdict("r1", "debt-reduction", "Keep going; drop DR3.");
    expect(calls).toEqual([
      { path: `${OPERATOR}/EnableSteering`, body: { commissionId: "c1" } },
      { path: `${OPERATOR}/AddVerdict`, body: { reviewId: "r1", topic: "debt-reduction", text: "Keep going; drop DR3." } },
    ]);
  });
});

describe("launch requests", () => {
  it("builds from the spec JSON, then launches automatically by commission id", async () => {
    const calls = stubWire();
    const spec = JSON.stringify({ topics: ["debt-reduction"], allowance_hours: 24, stance_quotes: ["q"] });
    await launchApi.buildFromSpec(spec);
    await launchApi.launch("research-20261111");
    expect(calls).toEqual([
      { path: `${LAUNCH}/Build`, body: { specJson: spec } },
      { path: `${LAUNCH}/Launch`, body: { commissionId: "research-20261111" } },
    ]);
  });

  it("carries one choice per blocking commission and the never-started assertion, never undecided ones (rulings P1, P2)", async () => {
    const calls = stubWire();
    await launchApi.launch("standing-teams-w1-20261110", {
      choices: { "programme-four-20261110": "supersede", "local-models-w1-20261111": "alongside", "research-night-20261031": "" },
      assertNeverStarted: true,
    });
    expect(calls).toEqual([
      {
        path: `${LAUNCH}/Launch`,
        body: {
          commissionId: "standing-teams-w1-20261110",
          resolutions: [
            { commissionId: "local-models-w1-20261111", choice: "RESOLUTION_RUN_ALONGSIDE" },
            { commissionId: "programme-four-20261110", choice: "RESOLUTION_SUPERSEDE" },
          ],
          assertNeverStarted: true,
        },
      },
    ]);
  });

  it("builds with the operator's choices so the charter names them (rulings P2)", async () => {
    const calls = stubWire();
    await launchApi.buildFromSpec("{}", { "programme-four-20261110": "supersede" });
    expect(calls).toEqual([{ path: `${LAUNCH}/Build`, body: { specJson: "{}", resolutions: [{ commissionId: "programme-four-20261110", choice: "RESOLUTION_SUPERSEDE" }] } }]);
  });

  it("relaunches from an earlier packet", async () => {
    const calls = stubWire();
    await launchApi.relaunch("programme-three-20261111");
    expect(calls[0]).toEqual({ path: `${LAUNCH}/Build`, body: { relaunchOf: "programme-three-20261111" } });
  });

  it("sends each control with its action, and a discard with its reason", async () => {
    const calls = stubWire();
    await launchApi.control("c1", ControlAction.PAUSE);
    await launchApi.control("c1", ControlAction.RESEND_STANCE);
    await launchApi.control("c1", ControlAction.DISCARD, "wrong topics");
    expect(calls.map((c) => [c.path, c.body])).toEqual([
      [`${LAUNCH}/Control`, { commissionId: "c1", action: "CONTROL_ACTION_PAUSE" }],
      [`${LAUNCH}/Control`, { commissionId: "c1", action: "CONTROL_ACTION_RESEND_STANCE" }],
      [`${LAUNCH}/Control`, { commissionId: "c1", action: "CONTROL_ACTION_DISCARD", reason: "wrong topics" }],
    ]);
  });
});
