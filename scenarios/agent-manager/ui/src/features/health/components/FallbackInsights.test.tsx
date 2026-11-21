import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { FallbackInsightsResponseSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { renderWithProviders } from "../../../test-utils/index.js";
import { FallbackInsights } from "./FallbackInsights.js";

const measures = vi.hoisted(() => ({ fallbackInsights: vi.fn() }));
vi.mock("../../usage/measuresClient.js", () => ({ measuresClient: measures }));

test("fallback insights summarise runner and model fallbacks with their pairs", async () => {
  measures.fallbackInsights.mockResolvedValue(create(FallbackInsightsResponseSchema, { runnerAttempts: 3n, runnerExhausted: 1n, modelAttempts: 2n, runnerPairs: [{ from: "codex", to: "claude-code", reason: "rate_limited", count: 3n }] }));
  renderWithProviders(createElement(FallbackInsights));
  const card = await screen.findByTestId("fallback-insights");
  await screen.findByText(/codex → claude-code · Rate limited · 3×/);
  assert.match(card.textContent ?? "", /Runner: 3 fallbacks, 1 ran out of options\. Model: 2, 0 ran out/);
  assert.equal(measures.fallbackInsights.mock.calls[0]?.[1].timeoutMs > 0, true);
});
