/**
 * Self-tests for the cross-domain proto-typed test factories.
 *
 * Factories are the load-bearing source of test data for every UI test
 * that exercises an API-shaped value. If a factory drifts away from the
 * generated proto descriptor (wrong default, missing field, broken
 * round-trip), the failure mode is silent: tests pass against fake data
 * that no longer matches what production code receives from the API.
 *
 * Domain-specific factories have their own self-tests next to the
 * feature (e.g. `features/notes/mocks/factories.test.ts`).
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

import { ResponseSchema } from "@vrooli/proto-types/personal-planner/v1/shared/health_pb";

import { makeHealthResponse } from "./factories";

describe("makeHealthResponse", () => {
  it("preserves deliberately invalid field overrides", () => {
    const value = makeHealthResponse({ status: "", timestamp: "not-a-date", readiness: false });
    expect(value.status).toBe("");
    expect(value.timestamp).toBe("not-a-date");
    expect(value.readiness).toBe(false);
  });

  it("allocates independent nested dependency maps and values", () => {
    const overrides = { dependencies: { database: { connected: true, database: "original" } } };
    const first = makeHealthResponse(overrides);
    const second = makeHealthResponse(overrides);
    const firstDatabase = first.dependencies.database;
    expect(firstDatabase).toBeDefined();
    if (!firstDatabase) throw new Error("database dependency should be present");
    firstDatabase.database = "changed";
    delete first.dependencies.database;
    expect(second.dependencies.database?.database).toBe("original");
    expect(overrides.dependencies.database.database).toBe("original");
  });

  it("copies already-created protobuf dependencies rather than sharing messages", () => {
    const source = makeHealthResponse({ dependencies: { database: { database: "source" } } });
    const copied = makeHealthResponse({ dependencies: source.dependencies });
    const copiedDatabase = copied.dependencies.database;
    expect(copiedDatabase).toBeDefined();
    if (!copiedDatabase) throw new Error("copied database dependency should be present");
    copiedDatabase.database = "changed";
    expect(source.dependencies.database?.database).toBe("source");
  });

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
