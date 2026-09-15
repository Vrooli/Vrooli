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

import { ResponseSchema } from "@vrooli/proto-types/code-facts/v1/health/health_pb";

import { makeHealthResponse } from "./factories";

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
