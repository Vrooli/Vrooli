import { describe, it, expect } from "vitest";
import {
  infrastructureHealthResponseSchema,
  knowledgeHealthResponseSchema,
  searchResponseSchema,
} from "./knowledgeContracts";

describe("knowledgeContracts", () => {
  it("accepts valid /health responses", () => {
    const payload = {
      status: "healthy",
      service: "knowledge-observatory",
      timestamp: "2026-01-25T10:00:00Z",
      readiness: true,
      version: "1.0.0",
      uptime_seconds: 123.4,
      dependencies: {
        postgres: { connected: true, latency_ms: 5.2 },
      },
      metrics: {
        goroutines: 42,
        heap_mb: 128.5,
      },
    };

    const result = infrastructureHealthResponseSchema.safeParse(payload);

    expect(result.success).toBe(true);
  });

  it("rejects invalid /health responses", () => {
    const result = infrastructureHealthResponseSchema.safeParse({
      status: "",
      service: "",
      timestamp: "",
      readiness: true,
    });

    expect(result.success).toBe(false);
  });

  it("accepts valid search responses", () => {
    const payload = {
      results: [
        {
          id: "r-1",
          score: 0.92,
          content: "Semantic result",
          metadata: { source: "alpha", score: 0.92 },
        },
      ],
      query: "semantic query",
      took_ms: 12,
    };

    const result = searchResponseSchema.safeParse(payload);

    expect(result.success).toBe(true);
  });

  it("rejects invalid search responses", () => {
    const result = searchResponseSchema.safeParse({
      results: "nope",
      query: 42,
      took_ms: "bad",
    });

    expect(result.success).toBe(false);
  });

  it("accepts valid knowledge health responses", () => {
    const payload = {
      total_entries: 42,
      collections: [
        {
          name: "alpha",
          size: 11,
          metrics: { coherence: 0.6, freshness: 0.7 },
        },
      ],
      overall_health: "steady",
      overall_metrics: { coherence: 0.6, freshness: 0.7 },
      timestamp: "2026-01-25T10:00:00Z",
    };

    const result = knowledgeHealthResponseSchema.safeParse(payload);

    expect(result.success).toBe(true);
  });

  it("rejects invalid knowledge health responses", () => {
    const result = knowledgeHealthResponseSchema.safeParse({
      total_entries: 42,
      collections: [{ name: "", size: 0 }],
      overall_health: "",
      timestamp: "",
    });

    expect(result.success).toBe(false);
  });
});
