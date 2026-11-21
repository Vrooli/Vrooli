import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { QueryClient } from "@tanstack/react-query";
import { SettingsPage } from "./SettingsPage";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { defaultApiClient } from "../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

// Real page, tabs, service translation, Connect clients and query cache.
// Only HTTP boundaries are controlled. No actual policy or runtime changes.
let owner = structuredClone(DEFAULT_SETTINGS);
let readRefused = false, integrationRefused = false, runRefused = false;
let audioRefused = false;
let voice = "owner-voice";
let summarizeEnabled = true, summarizeLevel = "SUMMARIZE_LEVEL_MODERATE", summarizeThreshold = 500;
let completeRun: ((value: Response) => void) | undefined;
const clients: QueryClient[] = [], unexpected: string[] = [];
let themeBefore: string | undefined, resolvedBefore: string | undefined, colorBefore = "";
const requests: { path: string; body: string }[] = [];
function response() { const { deleteConfirmation: _, ...settings } = owner; return { settings }; }
function json(value: unknown) { return new Response(JSON.stringify(value), { headers: { "Content-Type": "application/json" } }); }
function status(findings = 3) { return { enabled: true, mode: "suggest", strategy: "importance", candidates: 3, findings, created: 0, openAutoFiled: 2, maxOpenAutoFiled: 10, remainingBudget: 8, lastCycleTime: "2026-10-06T00:00:00Z" }; }
function mount() { const client = createTestQueryClient(); clients.push(client); return { ...renderWithProviders(<SettingsPage />, { queryClient: client }), client }; }
async function loaded() { await screen.findByRole("button", { name: "Save Settings" }); }
async function tab(name: string) { await loaded(); await userEvent.click(screen.getByRole("tab", { name })); }
function input(label: string) { const element = screen.getByText(label, { selector: "label" }).parentElement?.querySelector("input"); if (!element) throw new Error(`Missing real input ${label}`); return element; }
function change(label: string, value: string) { fireEvent.change(input(label), { target: { value } }); }
beforeEach(() => {
  themeBefore = document.documentElement.dataset.theme; resolvedBefore = document.documentElement.dataset.resolvedTheme; colorBefore = document.documentElement.style.colorScheme;
  audioRefused = false; voice = "owner-voice"; summarizeEnabled = true; summarizeLevel = "SUMMARIZE_LEVEL_MODERATE"; summarizeThreshold = 500; owner = structuredClone(DEFAULT_SETTINGS); readRefused = false; integrationRefused = false; runRefused = false; completeRun = undefined; unexpected.length = 0; requests.length = 0;
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { if (readRefused) throw new Error("Settings owner unavailable"); return response(); }
    if (path === "/integrations") { if (integrationRefused) throw new Error("Integration unavailable"); return { integrations: [{ id: "agent-manager", availability: "unconfigured", degradedBehavior: "Starts require owner preflight", affectedTransitions: ["execute"] }] }; }
    if (path === "/stats") return {};
    if (path === "/execution/auto-drain") return { enabled: false };
    unexpected.push(path); throw new Error(`Unexpected settings read ${path}`);
  });
  for (const method of ["put", "post", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (value: RequestInfo | URL, init?: RequestInit) => {
    const request = value instanceof Request ? new Request(value, init) : new Request(String(value), init); const path = new URL(request.url).pathname; requests.push({ path, body: await request.text() });
    if (path.endsWith("/ListVoices")) return json({ voices: [{ id: "owner-voice", name: "Owner voice" }, { id: "second-voice", name: "Second voice" }] });
    if (path.endsWith("/GetTTSConfig")) return json({ config: { defaultVoice: voice, defaultSpeed: 1, defaultResponseFormat: "RESPONSE_FORMAT_MP3" } });
    if (path.endsWith("/GetSummarizeConfig")) return json({ config: { enabled: summarizeEnabled, level: summarizeLevel, charThreshold: summarizeThreshold } });
    if (path.endsWith("/UpdateTTSConfig")) { if (audioRefused) return new Response(JSON.stringify({ code: "unavailable", message: "Audio owner unavailable" }), { status: 503, headers: { "Content-Type": "application/json" } }); const body = JSON.parse(requests[requests.length - 1]!.body); voice = body.config.defaultVoice ?? voice; return json({ config: { defaultVoice: voice, defaultSpeed: body.config.defaultSpeed ?? 1 } }); }
    if (path.endsWith("/UpdateSummarizeConfig")) { const body = JSON.parse(requests[requests.length - 1]!.body); if (body.updateMask === "enabled") summarizeEnabled = body.config.enabled ?? false; summarizeLevel = body.config.level ?? summarizeLevel; summarizeThreshold = body.config.charThreshold ?? summarizeThreshold; return json({ config: { enabled: summarizeEnabled, level: summarizeLevel, charThreshold: summarizeThreshold } }); }
    if (path.endsWith("/ListTransitions")) return json({ transitions: [] });
    if (path.endsWith("/GetStatus")) return json(status());
    if (path.endsWith("/RunNow")) { if (runRefused) return new Response(JSON.stringify({ code: "permission_denied", message: "Auto-filer owner refused" }), { status: 403, headers: { "Content-Type": "application/json" } }); return new Promise<Response>(resolve => { completeRun = resolve; }); }
    unexpected.push(path); throw new Error(`Unexpected settings transport ${path}`);
  }));
});
afterEach(() => { expect(unexpected).toEqual([]); for (const method of ["post", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); if (themeBefore === undefined) delete document.documentElement.dataset.theme; else document.documentElement.dataset.theme = themeBefore; if (resolvedBefore === undefined) delete document.documentElement.dataset.resolvedTheme; else document.documentElement.dataset.resolvedTheme = resolvedBefore; document.documentElement.style.colorScheme = colorBefore; });

describe("SettingsPage actual ownership workflows", () => {
  it("shows integration limits without enabling an unchanged write", async () => {
    mount(); await loaded(); expect(screen.getByText("Starts require owner preflight")).toBeVisible(); expect(screen.getByText("Affects: execute")).toBeVisible(); expect(screen.getByRole("button", { name: "Save Settings" })).toBeDisabled(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("keeps settings usable when advisory integrations fail", async () => {
    integrationRefused = true; mount(); await loaded(); expect(await screen.findByText(/Integration status is currently unavailable/, {}, { timeout: 12000 })).toBeVisible(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  }, 15000);
  it("refuses unavailable settings then explicitly retries only its read", async () => {
    readRefused = true; mount(); expect(await screen.findByText("Unable to load settings", {}, { timeout: 12000 })).toBeVisible(); expect(screen.queryByRole("button", { name: "Save Settings" })).toBeNull(); readRefused = false; fireEvent.click(screen.getByRole("button", { name: "Try again" })); await loaded(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  }, 15000);
  it("previews theme locally and restores saved theme on dirty unmount", async () => {
    const view = mount(); await loaded(); const before = view.client.getQueryData(["settings"]); fireEvent.click(screen.getByTestId(selectors.settings.themeLight)); expect(document.documentElement.dataset.resolvedTheme).toBe("light"); expect(view.client.getQueryData(["settings"])).toBe(before); expect(defaultApiClient.put).not.toHaveBeenCalled(); view.unmount(); expect(document.documentElement.dataset.resolvedTheme).toBe("dark");
  });
  it("waits for one exact typed write and canonical response before changing saved cache", async () => {
    let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.put).mockImplementation(() => new Promise(resolve => { complete = resolve; })); const { client } = mount(); await loaded(); const before = client.getQueryData(["settings"]); fireEvent.click(screen.getByTestId(selectors.settings.themeLight)); fireEvent.click(screen.getByRole("button", { name: "Save Settings" })); await waitFor(() => expect(defaultApiClient.put).toHaveBeenCalledOnce()); expect(vi.mocked(defaultApiClient.put).mock.calls[0]?.[0]).toBe("/settings"); expect(vi.mocked(defaultApiClient.put).mock.calls[0]?.[1]).toMatchObject({ theme: "light", auto_fixup: false, review_max_blocking_violations: 0, review_max_warnings: -1 }); expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Saving..." })); expect(defaultApiClient.put).toHaveBeenCalledOnce(); expect(client.getQueryData(["settings"])).toBe(before); owner = { ...owner, theme: "system" }; await act(async () => complete(response())); expect(await screen.findByText("Settings saved.")).toBeVisible(); expect(client.getQueryData(["settings"])).toMatchObject({ theme: "system" }); expect(screen.getByRole("button", { name: "Save Settings" })).toBeDisabled();
  });
  it("retains refused local edits and authoritative cache without automatic retry", async () => {
    vi.mocked(defaultApiClient.put).mockRejectedValue(new Error("Settings owner refused")); const { client } = mount(); await loaded(); const before = client.getQueryData(["settings"]); change("Search Debounce (ms)", "700"); fireEvent.click(screen.getByRole("button", { name: "Save Settings" })); expect(await screen.findByText("Failed to save settings")).toBeVisible(); expect(input("Search Debounce (ms)")).toHaveValue(700); expect(client.getQueryData(["settings"])).toBe(before); expect(defaultApiClient.put).toHaveBeenCalledOnce();
  });
  it.each([["Search Debounce (ms)", "0", 100], ["Search Debounce (ms)", "9000", 2000], ["Toast Duration (seconds)", "0", 1], ["Toast Duration (seconds)", "90", 30]] as const)("bounds %s input %s without saving", async (label, value, expected) => {
    mount(); await loaded(); change(label, value); expect(input(label)).toHaveValue(expected); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("resets edited UI preferences to the actual defaults without persistence", async () => {
    mount(); await loaded(); change("Search Debounce (ms)", "800"); change("Toast Duration (seconds)", "17"); fireEvent.click(within(screen.getByTestId(selectors.settings.uiPreferences)).getByRole("button", { name: "Reset" })); expect(input("Search Debounce (ms)")).toHaveValue(DEFAULT_SETTINGS.searchDebounceMs); expect(input("Toast Duration (seconds)")).toHaveValue(DEFAULT_SETTINGS.toastDurationMs / 1000); expect(screen.getByRole("button", { name: "Save Settings" })).toBeDisabled(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it.each([["Max Queue Depth", "0", 0], ["Circuit Breaker Threshold", "0", 1], ["Circuit Breaker Cooldown (minutes)", "9000", 1440], ["Cost Cap Per Run ($)", "-4", 0], ["Cost Per Turn Estimate ($)", "7", 5], ["Max Turns", "0", 5], ["Timeout (minutes)", "90", 60]] as const)("bounds execution %s input %s without dispatch", async (label, value, expected) => {
    mount(); await tab("Execution"); change(label, value); expect(input(label)).toHaveValue(expected); expect(defaultApiClient.put).not.toHaveBeenCalled(); expect(requests.some(request => request.path.endsWith("/RunNow"))).toBe(false);
  });
  it("keeps policy labeling while locally changing and resetting execution defaults", async () => {
    mount(); await tab("Execution"); const defaults = screen.getByTestId(selectors.settings.executionDefaults); expect(within(defaults).getByTestId("policy-controls-badge")).toBeVisible(); fireEvent.click(within(defaults).getByRole("button", { name: "yolo" })); expect(screen.getByRole("button", { name: "Save Settings" })).toBeEnabled(); fireEvent.click(within(defaults).getByRole("button", { name: "Reset" })); expect(screen.getByRole("button", { name: "Save Settings" })).toBeDisabled(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("waits for one explicit auto-filer request then presents the owner result", async () => {
    mount(); await tab("Execution"); fireEvent.click(screen.getByRole("button", { name: "Run now" })); await waitFor(() => expect(completeRun).toBeDefined()); expect(screen.getByRole("button", { name: "Run now" })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Run now" })); const runs = requests.filter(request => request.path.endsWith("/RunNow")); expect(runs).toHaveLength(1); expect(runs[0]?.body).toBe("{}"); await act(async () => completeRun?.(json(status(9)))); expect(await screen.findByText("9 / 0")).toBeVisible(); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("reports auto-filer refusal without a retry or alternate write", async () => {
    runRefused = true; mount(); await tab("Execution"); fireEvent.click(screen.getByRole("button", { name: "Run now" })); expect(await screen.findByText(/Auto-filer owner refused/)).toBeVisible(); expect(requests.filter(request => request.path.endsWith("/RunNow"))).toHaveLength(1); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it.each([["Minimum Code Quality Score", "190", 100], ["Minimum Test Pass Rate (%)", "-3", 0], ["Max Blocking Violations", "-5", 0], ["Max Warnings", "-6", -1]] as const)("bounds review %s input %s without applying policy", async (label, value, expected) => {
    mount(); await tab("Review"); change(label, value); expect(input(label)).toHaveValue(expected); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("preserves explicit false review requirements in the typed request", async () => {
    vi.mocked(defaultApiClient.put).mockResolvedValue(response()); mount(); await tab("Review"); const buttons = within(screen.getByTestId(selectors.settings.reviewSettings)).getAllByRole("button", { name: "Not Required" }); fireEvent.click(buttons[0]!); fireEvent.click(buttons[1]!); expect(screen.queryByText("Minimum Test Pass Rate (%)")).toBeNull(); fireEvent.click(screen.getByRole("button", { name: "Save Settings" })); await waitFor(() => expect(defaultApiClient.put).toHaveBeenCalledOnce()); expect(vi.mocked(defaultApiClient.put).mock.calls[0]?.[1]).toMatchObject({ review_require_tests: false, review_require_screenshots: false });
  });
  it("changes audio voice through its exact dedicated owner rather than scenario settings", async () => {
    mount(); await tab("Audio"); const select = await screen.findByTestId(selectors.settings.audioVoice); fireEvent.change(select, { target: { value: "second-voice" } }); await waitFor(() => expect(requests.filter(request => request.path.endsWith("/UpdateTTSConfig"))).toHaveLength(1)); const request = requests.find(request => request.path.endsWith("/UpdateTTSConfig")); expect(JSON.parse(request!.body)).toEqual({ updateMask: "defaultVoice", config: { defaultVoice: "second-voice" } }); await waitFor(() => expect(select).toHaveValue("second-voice")); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("retains the authoritative audio choice after a dedicated owner refusal", async () => {
    audioRefused = true; mount(); await tab("Audio"); const select = await screen.findByTestId(selectors.settings.audioVoice); fireEvent.change(select, { target: { value: "second-voice" } }); expect(await screen.findByText(/Audio-tools is unavailable/)).toBeVisible(); expect(select).toHaveValue("owner-voice"); expect(requests.filter(request => request.path.endsWith("/UpdateTTSConfig"))).toHaveLength(1); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("preserves explicit false summarization through the real update mask and canonical result", async () => {
    mount(); await tab("Audio"); const checkbox = await screen.findByRole("checkbox", { name: "Summarize long replies before TTS" }); expect(checkbox).toBeChecked(); fireEvent.click(checkbox); await waitFor(() => expect(checkbox).not.toBeChecked()); expect(requests.filter(request => request.path.endsWith("/UpdateSummarizeConfig"))).toHaveLength(1); expect(JSON.parse(requests.find(request => request.path.endsWith("/UpdateSummarizeConfig"))!.body)).toEqual({ updateMask: "enabled", config: {} }); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("sends the selected summarization level only to the audio owner", async () => {
    mount(); await tab("Audio"); const select = await screen.findByLabelText("Level"); fireEvent.change(select, { target: { value: "heavy" } }); await waitFor(() => expect(select).toHaveValue("heavy")); expect(JSON.parse(requests.find(request => request.path.endsWith("/UpdateSummarizeConfig"))!.body)).toEqual({ updateMask: "level", config: { level: "SUMMARIZE_LEVEL_HEAVY" } }); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("sends the explicit character threshold through its field mask", async () => {
    mount(); await tab("Audio"); const field = await screen.findByLabelText("Character threshold"); fireEvent.change(field, { target: { value: "900" } }); await waitFor(() => expect(field).toHaveValue(900)); expect(JSON.parse(requests.find(request => request.path.endsWith("/UpdateSummarizeConfig"))!.body)).toEqual({ updateMask: "charThreshold", config: { charThreshold: 900 } }); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });

  it("edits the complete auto-filer form locally and persists exact bounded values only on Save", async () => {
    vi.mocked(defaultApiClient.put).mockResolvedValue(response()); mount(); await tab("Execution"); const title = screen.getByRole("heading", { name: "Backlog Auto-Filer" }); const card = title.closest("[data-slot=card]") ?? title.parentElement?.parentElement?.parentElement; if (!card) throw new Error("Missing actual auto-filer card"); fireEvent.click(within(card as HTMLElement).getByRole("button", { name: "Enabled" })); fireEvent.click(await screen.findByRole("button", { name: "auto-add" })); fireEvent.click(screen.getByRole("button", { name: "importance" })); change("Max Open Auto-Filed", "5"); change("Velocity Window Days", "12"); change("Min Velocity Transitions", "3"); change("Interval Minutes", "20"); change("Goal Name", "owner-maintenance"); expect(defaultApiClient.put).not.toHaveBeenCalled(); fireEvent.click(screen.getByRole("button", { name: "Save Settings" })); await waitFor(() => expect(defaultApiClient.put).toHaveBeenCalledOnce()); expect(vi.mocked(defaultApiClient.put).mock.calls[0]?.[1]).toMatchObject({ auto_filer: { enabled: true, mode: "auto_add", strategy: "importance", max_open_auto_filed: 5, velocity_window_days: 12, min_velocity_transitions: 3, interval_minutes: 20, goal_name: "owner-maintenance" } }); expect(requests.some(request => request.path.endsWith("/RunNow"))).toBe(false);
  });
  it("resets agent defaults locally after edits rather than applying a run policy", async () => {
    mount(); await tab("Execution"); change("Max Turns", "77"); change("Timeout (minutes)", "7"); fireEvent.click(within(screen.getByTestId(selectors.settings.agentSettings)).getByRole("button", { name: "Reset" })); expect(input("Max Turns")).toHaveValue(DEFAULT_SETTINGS.agentMaxTurns); expect(input("Timeout (minutes)")).toHaveValue(DEFAULT_SETTINGS.agentTimeoutSeconds / 60); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });
  it("resets review thresholds after local edits without persisting them", async () => {
    mount(); await tab("Review"); change("Minimum Code Quality Score", "81"); change("Max Warnings", "2"); fireEvent.click(within(screen.getByTestId(selectors.settings.reviewSettings)).getByRole("button", { name: "Reset" })); expect(input("Minimum Code Quality Score")).toHaveValue(DEFAULT_SETTINGS.reviewCodeQualityMinScore); expect(input("Max Warnings")).toHaveValue(DEFAULT_SETTINGS.reviewMaxWarnings); expect(defaultApiClient.put).not.toHaveBeenCalled();
  });

});
