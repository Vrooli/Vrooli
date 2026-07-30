import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { ImportPage } from "../../src/pages/ImportPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const sources = { sources: [{ runnerType: "codex", label: "Codex", state: "ready", sessionCount: 1 }] };
const sessions = { sessions: [{ key: "2026/07/example.jsonl", sessionId: "session-1", title: "Fix the import flow", updatedAt: "2026-07-30T12:00:00Z" }] };
const reply = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

afterEach(() => vi.restoreAllMocks());

test("ImportPage browses a runner's saved sessions and imports selected evidence", async () => {
  const fetchMock = vi.fn().mockResolvedValueOnce(reply(sources)).mockResolvedValueOnce(reply(sessions)).mockResolvedValueOnce(reply({ id: "run-1" }, 201)).mockResolvedValueOnce(reply(sessions));
  vi.stubGlobal("fetch", fetchMock);
  renderWithProviders(createElement(ImportPage));
  assert.ok(await screen.findByText("Fix the import flow"));
  fireEvent.click(screen.getByLabelText("Select Fix the import flow"));
  fireEvent.click(screen.getByRole("button", { name: "Import selected (1)" }));
  assert.ok(await screen.findByText("1 conversation imported as read-only runs."));
  assert.ok(String(fetchMock.mock.calls[2][0]).endsWith("/api/v1/import/sessions"));
});

test("ImportPage makes the existing association visible and not selectable", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(reply(sources)).mockResolvedValueOnce(reply({ sessions: [{ ...sessions.sessions[0], importedRunId: "run-1" }] })));
  renderWithProviders(createElement(ImportPage));
  assert.ok(await screen.findByText("Already imported"));
  assert.equal((screen.getByLabelText("Select Fix the import flow") as HTMLInputElement).disabled, true);
});

test("ImportPage explains runner discovery failure", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("no", { status: 503 })));
  renderWithProviders(createElement(ImportPage));
  assert.ok(await screen.findByText("Runner sessions could not be loaded. Check the runner resource and try again."));
});
