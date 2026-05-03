/**
 * Self-tests for the proto-typed test factories.
 *
 * Factories are the load-bearing source of test data for every UI test
 * that exercises an API-shaped value. If a factory drifts away from the
 * generated proto descriptor (wrong default, missing field, broken
 * round-trip), the failure mode is silent: tests pass against fake data
 * that no longer matches what production code receives from the API.
 *
 * The shape pinned below mirrors the Go-side `fixtures/health_test.go`:
 *
 *   - sane defaults make the most common test path `makeX()` no-args
 *   - overrides merge field-level (no all-or-nothing replacement)
 *   - the returned instance round-trips through proto's
 *     `toJson` / `fromJson` byte-identically — i.e., it includes the
 *     internal `$typeName`/reflection state proto runtime needs
 */
import { fromJson, toJson } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";

import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";
import {
  ListNotesResponseSchema,
  NoteSchema,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/notes/notes_pb";

import {
  makeHealthResponse,
  makeListNotesResponse,
  makeNote,
} from "./factories";

describe("makeHealthResponse", () => {
  it("returns a healthy default with non-empty service/version/timestamp", () => {
    const r = makeHealthResponse();
    expect(r.status).toBe("healthy");
    expect(r.readiness).toBe(true);
    expect(r.service).not.toBe("");
    expect(r.version).not.toBe("");
    expect(r.timestamp).not.toBe("");
  });

  it("default timestamp is parseable as a Date", () => {
    const r = makeHealthResponse();
    const d = new Date(r.timestamp);
    expect(Number.isNaN(d.getTime())).toBe(false);
  });

  it("merges overrides field-by-field, leaving unset fields at defaults", () => {
    const r = makeHealthResponse({ status: "degraded", version: "9.9.9" });
    expect(r.status).toBe("degraded");
    expect(r.version).toBe("9.9.9");
    // unspecified fields keep the factory defaults
    expect(r.readiness).toBe(true);
    expect(r.service).not.toBe("");
  });

  it("round-trips through proto JSON encode + decode byte-identically", () => {
    // The round-trip is the contract: production code consumes this
    // shape via `fromJson(ResponseSchema, ...)` over the wire. If the
    // factory ever produces a value that doesn't survive the round-trip,
    // tests against the factory pass but production breaks on real
    // responses.
    const original = makeHealthResponse({ status: "degraded" });
    const json = toJson(ResponseSchema, original);
    const decoded = fromJson(ResponseSchema, json);
    expect(decoded.status).toBe("degraded");
    expect(decoded.service).toBe(original.service);
    expect(decoded.timestamp).toBe(original.timestamp);
    expect(decoded.readiness).toBe(original.readiness);
  });
});

describe("makeNote", () => {
  it("returns a note with non-empty id/title and RFC3339 timestamps", () => {
    const n = makeNote();
    expect(n.id).not.toBe("");
    expect(n.title).not.toBe("");
    expect(Number.isNaN(new Date(n.createdAt).getTime())).toBe(false);
    expect(Number.isNaN(new Date(n.updatedAt).getTime())).toBe(false);
  });

  it("merges overrides without dropping defaults", () => {
    const n = makeNote({ id: "custom-1", title: "Custom" });
    expect(n.id).toBe("custom-1");
    expect(n.title).toBe("Custom");
    expect(n.createdAt).not.toBe("");
  });

  it("round-trips through NoteSchema JSON encode + decode", () => {
    const original = makeNote({ id: "rt-1", title: "round trip" });
    const decoded = fromJson(NoteSchema, toJson(NoteSchema, original));
    expect(decoded.id).toBe("rt-1");
    expect(decoded.title).toBe("round trip");
    expect(decoded.createdAt).toBe(original.createdAt);
  });
});

describe("makeListNotesResponse", () => {
  it("defaults to an empty notes array (proto3: maps/arrays default to [], not undefined)", () => {
    const r = makeListNotesResponse();
    expect(r.notes).toEqual([]);
  });

  it("accepts an overrides.notes array of factory-built Notes", () => {
    const r = makeListNotesResponse({
      notes: [makeNote({ id: "a" }), makeNote({ id: "b" })],
    });
    expect(r.notes).toHaveLength(2);
    expect(r.notes[0]?.id).toBe("a");
    expect(r.notes[1]?.id).toBe("b");
  });

  it("round-trips through ListNotesResponseSchema with embedded notes", () => {
    const original = makeListNotesResponse({
      notes: [makeNote({ id: "rt-list-1", title: "embedded" })],
    });
    const decoded = fromJson(
      ListNotesResponseSchema,
      toJson(ListNotesResponseSchema, original),
    );
    expect(decoded.notes).toHaveLength(1);
    expect(decoded.notes[0]?.id).toBe("rt-list-1");
    expect(decoded.notes[0]?.title).toBe("embedded");
  });
});
