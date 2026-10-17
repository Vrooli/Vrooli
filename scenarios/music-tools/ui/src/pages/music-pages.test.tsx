import { webcrypto, createHash } from "node:crypto";
import { Blob as NodeBlob, Buffer } from "node:buffer";
import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { AuditionPage } from "./AuditionPage";
import { CompositionPage } from "./CompositionPage";
import { PoolPage } from "./PoolPage";
import { StylesPage } from "./StylesPage";

const take = { id: "take-1", style_id: "launch-trap", blob_ref: "/audio.wav", provenance: { seed: 4, applied_rung: "offload-dit" } };
const fetchMock = vi.fn();
let created: NodeBlob[];


beforeEach(() => {
  localStorage.clear();
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => undefined);
  created = [];
  // JSDOM and Node use different ArrayBuffer realms. Bridge the fixture
  // bytes into Node's native crypto while retaining the real SHA-256 algorithm.
  vi.stubGlobal("crypto", { subtle: { digest: (algorithm: string, data: ArrayBuffer) => webcrypto.subtle.digest(algorithm, Buffer.from(new Uint8Array(data))) } });
  vi.stubGlobal("Blob", NodeBlob);
  vi.stubGlobal("URL", class extends URL {
    static createObjectURL = vi.fn((blob: Blob | MediaSource) => { created.push(blob as unknown as NodeBlob); return `blob:audition-${created.length}`; });
    static revokeObjectURL = vi.fn();
  });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
  fetchMock.mockReset();
  fetchMock.mockImplementation((url: string) => {
    if (url.endsWith("/audio")) return Promise.resolve({ ok: true, arrayBuffer: async () => Uint8Array.from([1,2,3,4]).buffer });
    if (url.includes("styles")) return Promise.resolve({ ok: true, json: async () => [{ id: "launch-trap", name: "Launch Trap", caption: "dark trap" }] });
    if (url.includes("pool/status")) return Promise.resolve({ ok: true, json: async () => ({ available: 2, reserved: 1, target_depth: 10, replenish_below: 3 }) });
    if (url.includes("pool/draw")) return Promise.resolve({ ok: true, json: async () => ({ take }) });
    if (url.includes("compose")) return Promise.resolve({ ok: true, json: async () => ({ job_id: "job-1" }) });
    if (url.includes("takes")) return Promise.resolve({ ok: true, json: async () => ({ takes: [take] }) });
    return Promise.resolve({ ok: true, json: async () => ({}) });
  });
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe("music feature pages", () => {
  it("renders styles and fetched style data", async () => { renderWithProviders(<StylesPage />); expect(await screen.findByText("Launch Trap")).toBeInTheDocument(); });
  it("renders audition peers with in-place playback", async () => { renderWithProviders(<AuditionPage />); expect(await screen.findByLabelText("Play take-1")).toBeInTheDocument(); });
  it("records listening separately from selection and retains sources", async () => {
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(player).toHaveAttribute("src", "blob:audition-1"));
    expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/api\/v1\/takes\/take-1\/audio$/), expect.objectContaining({ signal: expect.any(AbortSignal) }));
    fireEvent.click(screen.getByRole("button", { name: "Keep for this cut" }));
    expect(screen.getByRole("button", { name: "Keep for this cut" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("checkbox")).not.toBeChecked();
    expect(fetchMock.mock.calls.every(([, options]) => !options?.method || options.method === "GET")).toBe(true);
  });
  it("does not export a listening claim without operator attestation", async () => {
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Enter your name and confirm");
  });
  it("binds exported findings to the same audio bytes used for playback", async () => {
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Fixture listener" } });
    fireEvent.change(screen.getByLabelText("Listening notes take-1"), { target: { value: "Energetic groove; ending needs a recut" } });
    fireEvent.click(screen.getByRole("button", { name: "Reject for this cut" }));
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    await waitFor(() => expect(created).toHaveLength(2));
    const review = JSON.parse(await created[1]!.text());
    expect(review.reviewer).toBe("Fixture listener");
    expect(review.takes[0]).toMatchObject({ choice: "reject", listened: true, notes: "Energetic groove; ending needs a recut", sha256: createHash("sha256").update(Uint8Array.from([1,2,3,4])).digest("hex") });
    expect(fetchMock.mock.calls.filter(([url]) => url.endsWith("/audio"))).toHaveLength(1);
  });
  it("surfaces an audio fetch failure without enabling listening attestation", async () => {
    fetchMock.mockImplementation((url: string) => Promise.resolve(url.endsWith("/audio") ? { ok: false, status: 503 } : { ok: true, json: async () => ({ takes: [take] }) }));
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Audio load failed");
    expect(screen.getByRole("checkbox")).toBeDisabled();
  });
  it("refuses oversized audio before buffering or enabling listening attestation", async () => {
    const buffer = vi.fn();
    fetchMock.mockImplementation((url: string) => Promise.resolve(url.endsWith("/audio")
      ? { ok: true, headers: { get: () => String(65 * 1024 * 1024) }, arrayBuffer: buffer }
      : { ok: true, json: async () => ({ takes: [take] }) }));
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("exceeds the browser budget");
    expect(buffer).not.toHaveBeenCalled();
    expect(screen.getByRole("checkbox")).toBeDisabled();
  });
  it("renders pool depth", async () => { renderWithProviders(<PoolPage />); expect(await screen.findByText(/2 available/)).toBeInTheDocument(); });
  it("queues composition and draws a take", async () => {
    renderWithProviders(<CompositionPage />);
    fireEvent.click(screen.getByRole("button", { name: "Compose" }));
    fireEvent.click(screen.getByRole("button", { name: "Draw take" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(screen.getByText(/seed 4/)).toBeInTheDocument();
  });
  it("unloads playback bytes while preserving hash-bound findings and saved draft", async () => {
    const first = renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Listener" } });
    fireEvent.change(screen.getByLabelText("Cut or project"), { target: { value: "Fresh launch" } });
    fireEvent.click(screen.getByRole("button", { name: "Keep for this cut" }));
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Unload audio take-1" }));
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:audition-1");
    expect(screen.getByRole("checkbox")).toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Save review draft" }));
    first.unmount();
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Restore review draft" }));
    expect(screen.getByLabelText("Reviewer")).toHaveValue("Listener");
    expect(screen.getByRole("checkbox")).toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    await waitFor(() => expect(created).toHaveLength(2));
    const review = JSON.parse(await created[1]!.text());
    expect(review.cut).toBe("Fresh launch");
    expect(review.takes[0]).toMatchObject({ choice: "keep", listened: true });
    expect(fetchMock.mock.calls.filter(([url]) => url.endsWith("/audio"))).toHaveLength(1);
  });
  it("invalidates attestation after a loaded source fails playback", async () => {
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.error(player);
    expect(screen.getByRole("checkbox")).not.toBeChecked();
    expect(screen.getByRole("alert")).toHaveTextContent("Playback failed");
  });
  it("uses a bounded job query and aborts its request on unmount", async () => {
    const result = renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.change(screen.getByLabelText("Job filter"), { target: { value: "fresh job" } });
    fireEvent.click(screen.getByRole("button", { name: "Load candidate batch" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("limit=20&job_id=fresh%20job"), expect.anything()));
    const request = fetchMock.mock.calls.at(-1)![1] as RequestInit;
    result.unmount();expect(request.signal?.aborted).toBe(true);
  });

  it.each([false, true])("reconciles a restored finding against cached bytes (matching=%s)", async (matching) => {
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    const actual = createHash("sha256").update(Uint8Array.from([1,2,3,4])).digest("hex");
    const restored = matching ? actual : "b".repeat(64);
    localStorage.setItem("music-audition-draft-v1", JSON.stringify({
      version: 1, reviewer: "Listener", cut: "Fresh cut",
      findings: { "take-1": { choice: "keep", listened: true, notes: "Reviewed this identity" } },
      reviewSources: { "take-1": { sha256: restored, take } },
    }));
    fireEvent.click(screen.getByRole("button", { name: "Restore review draft" }));
    if (matching) {
      expect(player).toHaveAttribute("src", "blob:audition-1");
      expect(screen.getByRole("checkbox")).toBeChecked();
      fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
      await waitFor(() => expect(created).toHaveLength(2));
      const review = JSON.parse(await created[1]!.text());
      expect(review.takes[0].sha256).toBe(actual);
    } else {
      expect(player).not.toHaveAttribute("src");
      expect(screen.getByRole("checkbox")).toBeDisabled();
      expect(screen.getByRole("checkbox")).not.toBeChecked();
      expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:audition-1");
      fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
      expect(screen.getByRole("alert")).toHaveTextContent("confirm at least one listening review");
      expect(created).toHaveLength(1);
    }
  });

  it("keeps one peer playing, unloads all bytes and clears a completed review session", async () => {
    const peer = { ...take, id: "take-2" };
    fetchMock.mockImplementation((url: string) => Promise.resolve(url.endsWith("/audio")
      ? new Response(Uint8Array.from([1, 2, 3, 4]), { headers: { "Content-Type": "audio/wav" } })
      : { ok: true, json: async () => ({ takes: [take, peer] }) }));
    renderWithProviders(<AuditionPage />);
    const first = await screen.findByLabelText("Play take-1");
    const second = screen.getByLabelText("Play take-2");
    for (const id of ["take-1", "take-2"]) {
      fireEvent.click(screen.getByRole("button", { name: `Load audio ${id}` }));
      await waitFor(() => expect(screen.getByRole("button", { name: `Unload audio ${id}` })).toBeEnabled());
    }
    vi.mocked(HTMLMediaElement.prototype.pause).mockClear();
    fireEvent.play(first);
    expect(HTMLMediaElement.prototype.pause).toHaveBeenCalledTimes(1);
    expect(vi.mocked(HTMLMediaElement.prototype.pause).mock.contexts[0]).toBe(second);
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Peer listener" } });
    fireEvent.click(screen.getAllByRole("button", { name: "Keep for this cut" })[0]!);
    fireEvent.click(screen.getAllByRole("checkbox")[0]!);
    fireEvent.click(screen.getByRole("button", { name: "Unload all audio" }));
    expect(first).not.toHaveAttribute("src");
    expect(second).not.toHaveAttribute("src");
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:audition-1");
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:audition-2");
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    await waitFor(() => expect(created).toHaveLength(3));
    expect(JSON.parse(await created[2]!.text()).takes).toEqual([expect.objectContaining({ take_id: "take-1", listened: true })]);
    fireEvent.click(screen.getByRole("button", { name: "Start empty review session" }));
    expect(screen.getAllByRole("checkbox")[0]).not.toBeChecked();
    expect(screen.getAllByRole("button", { name: "Keep for this cut" })[0]).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    expect(screen.getByRole("alert")).toHaveTextContent("confirm at least one listening review");
    expect(created).toHaveLength(3);
  });

  it("aborts an in-flight candidate when starting empty and ignores its late bytes", async () => {
    let finish!: (bytes: ArrayBuffer) => void;
    const delayed = new Promise<ArrayBuffer>(resolve => { finish = resolve; });
    const digest = vi.fn((algorithm: string, data: ArrayBuffer) => webcrypto.subtle.digest(algorithm, Buffer.from(new Uint8Array(data))));
    vi.stubGlobal("crypto", { subtle: { digest } });
    let audioSignal: AbortSignal | undefined;
    fetchMock.mockImplementation((url: string, options: RequestInit) => {
      if (url.endsWith("/audio")) {
        audioSignal = options.signal as AbortSignal;
        return Promise.resolve({ ok: true, arrayBuffer: () => delayed });
      }
      return Promise.resolve({ ok: true, json: async () => ({ takes: [take] }) });
    });
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(audioSignal).toBeDefined());
    fireEvent.click(screen.getByRole("button", { name: "Start empty review session" }));
    expect(audioSignal!.aborted).toBe(true);
    await act(async () => { finish(Uint8Array.from([5, 6]).buffer); await delayed; });
    await waitFor(() => expect(digest).toHaveBeenCalledOnce());
    await act(async () => { await digest.mock.results[0]!.value; });
    expect(screen.getByRole("button", { name: "Load audio take-1" })).toBeEnabled();
    expect(player).not.toHaveAttribute("src");
    expect(screen.getByRole("checkbox")).toBeDisabled();
    expect(URL.createObjectURL).not.toHaveBeenCalled();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("requires a fresh listening review when reloaded source bytes change", async () => {
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Listener" } });
    fireEvent.change(screen.getByLabelText("Listening notes take-1"), { target: { value: "Original groove" } });
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Unload audio take-1" }));
    const changed = Uint8Array.from([8, 9, 10]);
    fetchMock.mockResolvedValueOnce({ ok: true, arrayBuffer: async () => changed.buffer });
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    expect(screen.getByRole("checkbox")).not.toBeChecked();
    expect(screen.getByLabelText("Listening notes take-1")).toHaveValue("Original groove\nSource bytes changed; listen again before exporting.");
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    expect(screen.getByRole("alert")).toHaveTextContent("confirm at least one listening review");
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    await waitFor(() => expect(created).toHaveLength(3));
    expect(JSON.parse(await created[2]!.text()).takes[0]).toMatchObject({ listened: true, sha256: createHash("sha256").update(changed).digest("hex") });
  });

  it.each([
    ["absent", null, "No supported draft"],
    ["invalid JSON", "{", "JSON"],
    ["unsupported version", JSON.stringify({ version: 2 }), "Draft is invalid"],
    ["candidate overflow", JSON.stringify({ version: 1, reviewer: "Listener", cut: "cut", findings: {}, reviewSources: Object.fromEntries(Array.from({ length: 101 }, (_, i) => [`take-${i}`, {}])) }), "candidate limit"],
    ["invalid source identity", JSON.stringify({ version: 1, reviewer: "Listener", cut: "cut", findings: {}, reviewSources: { "take-1": { sha256: "not-a-hash", take } } }), "invalid source findings"],
  ])("refuses %s drafts without replacing the active review", async (_label, draft, message) => {
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Current listener" } });
    fireEvent.click(screen.getByRole("button", { name: "Keep for this cut" }));
    fireEvent.click(screen.getByRole("checkbox"));
    if (draft !== null) localStorage.setItem("music-audition-draft-v1", draft!);
    fireEvent.click(screen.getByRole("button", { name: "Restore review draft" }));
    expect(screen.getByRole("alert")).toHaveTextContent(message!);
    expect(screen.getByLabelText("Reviewer")).toHaveValue("Current listener");
    expect(screen.getByRole("checkbox")).toBeChecked();
    expect(player).toHaveAttribute("src", "blob:audition-1");
  });

  it("preserves a listening review when browser draft storage is unavailable", async () => {
    renderWithProviders(<AuditionPage />);
    await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    fireEvent.change(screen.getByLabelText("Reviewer"), { target: { value: "Listener" } });
    fireEvent.click(screen.getByRole("button", { name: "Keep for this cut" }));
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.change(screen.getByLabelText("Listening notes take-1"), { target: { value: "n".repeat(1024 * 1024) } });
    fireEvent.click(screen.getByRole("button", { name: "Save review draft" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Draft could not be saved");
    expect(localStorage.getItem("music-audition-draft-v1")).toBeNull();
    fireEvent.change(screen.getByLabelText("Listening notes take-1"), { target: { value: "Retained groove review" } });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("Quota exceeded", "QuotaExceededError"); });
    fireEvent.click(screen.getByRole("button", { name: "Save review draft" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Draft could not be saved");
    fireEvent.click(screen.getByRole("button", { name: "Export listening review" }));
    await waitFor(() => expect(created).toHaveLength(2));
    expect(JSON.parse(await created[1]!.text()).takes[0]).toMatchObject({ listened: true, choice: "keep" });
  });

  it("shows list failure and recovers by loading a new candidate batch", async () => {
    fetchMock.mockResolvedValueOnce({ ok: false, status: 503 });
    renderWithProviders(<AuditionPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Take list failed (503)");
    expect(screen.queryByTestId("take-row")).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Job filter"), { target: { value: "recovery" } });
    fireEvent.click(screen.getByRole("button", { name: "Load candidate batch" }));
    expect(await screen.findByLabelText("Play take-1")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenLastCalledWith(expect.stringContaining("job_id=recovery"), expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("renders an older candidate without provenance and keeps unloaded playback errors harmless", async () => {
    fetchMock.mockResolvedValueOnce({ ok: true, json: async () => [{ id: "legacy-take" }] });
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play legacy-take");
    expect(screen.getByTestId("take-provenance")).toHaveTextContent("Seed — · Prompt not recorded");
    expect(screen.getByTestId("take-rung-badge")).toHaveTextContent("rung —");
    fireEvent.error(player);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox")).toBeDisabled();
  });

  it("submits the operator's selected style and peer count", async () => {
    renderWithProviders(<CompositionPage />);
    await screen.findByText(/seed 4/);
    fireEvent.change(screen.getByTestId("compose-style").querySelector("input")!, { target: { value: "cinematic pulse" } });
    fireEvent.change(screen.getByTestId("compose-take-count").querySelector("input")!, { target: { value: "3" } });
    expect(screen.getByTestId("compose-effective-caption")).toHaveTextContent("cinematic pulse");
    fireEvent.click(screen.getByRole("button", { name: "Compose" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/compose$/), expect.objectContaining({ method: "POST", body: JSON.stringify({ style_id: "cinematic pulse", takes: 3 }) })));
    expect(await screen.findByText(/Job job-1 queued/)).toBeInTheDocument();
  });

  it("refreshes readiness for the selected pool style without drawing a take", async () => {
    let available = 1;
    fetchMock.mockImplementation(() => Promise.resolve({ ok: true, json: async () => ({ available, reserved: 0, target_depth: 4, replenish_below: 2 }) }));
    renderWithProviders(<PoolPage />);
    expect(await screen.findByText("Replenishment status: below threshold")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Style"), { target: { value: "bright & fast" } });
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith(expect.stringContaining("style_id=bright%20%26%20fast")));
    available = 3;
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(await screen.findByText("Replenishment status: healthy")).toBeInTheDocument();
    expect(screen.getByText("3 available · 0 reserved · target 4")).toBeInTheDocument();
    expect(fetchMock.mock.calls.every(([url]) => url.includes("/pool/status?"))).toBe(true);
  });

  it("enforces the 100-candidate session boundary after restoring a load-only draft", async () => {
    renderWithProviders(<AuditionPage />);
    const player = await screen.findByLabelText("Play take-1");
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    await waitFor(() => expect(screen.getByRole("checkbox")).toBeEnabled());
    const sources = Object.fromEntries(Array.from({ length: 100 }, (_, i) => {
      const id = `draft-take-${i}`;
      return [id, { sha256: "b".repeat(64), take: { id } }];
    }));
    localStorage.setItem("music-audition-draft-v1", JSON.stringify({ version: 1, reviewer: "Prior listener", cut: "Prior cut", findings: {}, reviewSources: sources }));
    fireEvent.click(screen.getByRole("button", { name: "Restore review draft" }));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(player).not.toHaveAttribute("src");
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:audition-1");
    expect(screen.getByRole("checkbox")).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Load audio take-1" }));
    expect(screen.getByRole("alert")).toHaveTextContent("100 candidates");
    fireEvent.click(screen.getByRole("button", { name: "Keep for this cut" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Review session limit reached");
    expect(screen.getByRole("button", { name: "Keep for this cut" })).toHaveAttribute("aria-pressed", "false");
    expect(fetchMock.mock.calls.filter(([url]) => url.endsWith("/audio"))).toHaveLength(1);
  });

});
