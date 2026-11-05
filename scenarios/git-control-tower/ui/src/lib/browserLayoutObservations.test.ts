import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readBrowserLayout, startBrowserLayoutObservations } from "./browserLayoutObservations";
const record = vi.hoisted(() => vi.fn().mockResolvedValue({}));
vi.mock("./connect", () => ({ browserObservationClient: { recordBrowserObservation: record } }));
describe("automatic mobile client layout evidence", () => {
  let stop: (() => void) | undefined;
  beforeEach(() => {
    vi.useFakeTimers(); record.mockClear();
    Object.defineProperty(document, "elementFromPoint", { configurable: true, value: () => document.body });
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 393 });
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 828 });
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
    Object.defineProperty(navigator, "standalone", { configurable: true, value: true });
    document.body.innerHTML = '<main class="gct-mobile-shell"><header data-testid="mobile-header"></header><nav data-testid="mobile-nav"><div data-rcl-bottom-nav-track></div></nav></main>';
  });
  afterEach(() => { stop?.(); stop = undefined; vi.useRealTimers(); vi.restoreAllMocks(); document.body.replaceChildren(); });
  it("reports client dimensions and overlay transitions without page contents", async () => {
    stop = startBrowserLayoutObservations(); await vi.advanceTimersByTimeAsync(1100);
    expect(record).toHaveBeenCalledTimes(1);
    const first = record.mock.calls[0]?.[0].observation;
    expect(first.standalone).toBe(true);
    expect(JSON.parse(first.layoutJson).viewport).toEqual({ width: 393, height: 828 });
    expect(JSON.parse(first.layoutJson).overlays).toEqual([]);
    const drawer = document.createElement('section');
    drawer.setAttribute('data-rcl-full-page-drawer',''); drawer.setAttribute('data-state','open'); drawer.setAttribute('data-testid','settings');
    drawer.textContent = 'private repository contents'; document.body.append(drawer);
    await vi.advanceTimersByTimeAsync(1100);
    expect(record).toHaveBeenCalledTimes(2);
    const second = record.mock.calls[1]?.[0].observation;
    expect(JSON.parse(second.layoutJson).overlays[0].id).toBe('settings');
    expect(second.layoutJson).not.toContain('private repository contents');
    expect(second.clientId).toBe(first.clientId);
    drawer.remove(); await vi.advanceTimersByTimeAsync(1100);
    expect(JSON.parse(record.mock.calls[2]?.[0].observation.layoutJson).overlays).toEqual([]);
  });
  it("deduplicates unchanged readings and stops on cleanup", async () => {
    stop = startBrowserLayoutObservations(); await vi.advanceTimersByTimeAsync(1100);
    window.dispatchEvent(new Event('resize')); await vi.advanceTimersByTimeAsync(1100);
    expect(record).toHaveBeenCalledTimes(1);
    stop(); stop = undefined; window.dispatchEvent(new Event('resize')); await vi.advanceTimersByTimeAsync(1100);
    expect(record).toHaveBeenCalledTimes(1);
  });
  it("does not infer native pixels from browser geometry", () => {
    const probe = document.createElement('div'); for (let i=0;i<4;i++) probe.append(document.createElement('span')); document.body.append(probe);
    expect(readBrowserLayout(probe)).not.toHaveProperty('screenshot');
  });
});
