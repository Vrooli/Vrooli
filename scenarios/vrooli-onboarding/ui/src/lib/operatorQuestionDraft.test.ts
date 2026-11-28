import { describe, expect, it } from "vitest";
import { decodeOperatorQuestionDraft, encodeOperatorQuestionDraft } from "./operatorQuestionDraft";

describe("operator question drafts", () => {
  it("persists changed non-secret answers and declines without exposing question IDs", () => {
    const questions = [
      { id: "install.mode", kind: "enum", defaultValue: "local" },
      { id: "host.path", kind: "path", defaultValue: "/tmp" },
      { id: "host.confirm", kind: "confirmation", defaultValue: false },
    ];
    const choices = encodeOperatorQuestionDraft(questions, {
      "install.mode": "managed",
      "host.path": "/opt/vrooli",
      "host.confirm": true,
    }, { "host.confirm": true });

    expect(Object.keys(choices)).not.toContain("install.mode");
    expect(Object.keys(choices).every((key) => !/(credential|password|passphrase|secret|token|private_key)/i.test(key))).toBe(true);
    expect(decodeOperatorQuestionDraft(questions, choices)).toEqual({
      values: { "install.mode": "managed", "host.path": "/opt/vrooli", "host.confirm": true },
      declined: { "install.mode": false, "host.path": false, "host.confirm": true },
    });
  });

  it("never serializes or restores secret answers", () => {
    const questions = [
      { id: "credential-store.passphrase", kind: "secret" },
      { id: "host.accept", kind: "confirm", defaultValue: false },
    ];
    const choices = encodeOperatorQuestionDraft(questions, {
      "credential-store.passphrase": "fake-secret-value",
      "host.accept": true,
    }, {});

    expect(Object.values(choices)).not.toContain("fake-secret-value");
    expect(decodeOperatorQuestionDraft(questions, choices).values).toEqual({ "host.accept": true });
  });

  it("restores a deliberate empty answer over a non-empty default", () => {
    const questions = [{ id: "setup.name", kind: "choice", defaultValue: "recommended" }];
    const choices = encodeOperatorQuestionDraft(questions, { "setup.name": "" }, {});
    expect(Object.values(choices)).toEqual([""]);
    expect(decodeOperatorQuestionDraft(questions, choices).values).toEqual({ "setup.name": "" });
  });
});
