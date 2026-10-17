import { beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { ExplorePage } from "./ExplorePage";

const exploreRecipes = vi.hoisted(() => vi.fn());
const getPlan = vi.hoisted(() => vi.fn());
const applyPlan = vi.hoisted(() => vi.fn());
const previewSwap = vi.hoisted(() => vi.fn());
const ensureWorkspace = vi.hoisted(() => vi.fn());
vi.mock("../../api/planning", () => ({ exploreRecipes, getPlan, applyPlan, previewSwap }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace }));

describe("ExplorePage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    Object.defineProperty(navigator, "onLine", { configurable: true, value: true });
    ensureWorkspace.mockResolvedValue({ id: "w1" });
    exploreRecipes.mockResolvedValue({ planRevision: 4n, profileRevision: 2n, profileConfigured: true, candidates: [{ recipeId: "r1", name: "Lentil soup", recipeRevision: 3n, fitReasons: [{ code: "eligible", rule: "allergies", reference: "recipe.allergens.peanut", message: "Meets your saved food rules and kitchen requirements." }], summary: "Warm and simple." }] });
    getPlan.mockResolvedValue({ hasPlan: false, draft: { occurrences: [], unresolved: [], inputReferences: ["workspace:w1", "profile:2"], runId: "existing", seed: 0, currentRevision: 4n } });
    applyPlan.mockResolvedValue({ revision: 5n });
  });

  it("shows only server-evaluated candidates and retains slot context in recipe links", async () => {
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner"] });
    expect(await screen.findByRole("heading", { name: "Lentil soup" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Recipe photo unavailable" })).toBeInTheDocument();
    expect(screen.getByText("Planning Thursday dinner")).toBeInTheDocument();
    expect(screen.getByText("Meets your saved food rules and kitchen requirements.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open recipe" })).toHaveAttribute("href", expect.stringContaining("return=%2Fmeals%2Fexplore%3Fdate%3D2026-10-08%26slot%3Ddinner"));
  });

  it("does not claim fit when setup has not been applied", async () => {
    exploreRecipes.mockResolvedValue({ planRevision: 0n, profileRevision: 0n, profileConfigured: false, candidates: [{ recipeId: "r1", name: "Saved soup", recipeRevision: 1n, fitReasons: [{ code: "setup_unconfigured", rule: "setup", reference: "workspace.profile", message: "No setup rules are saved yet; this result is checked only against declared recipe evidence." }], summary: "" }] });
    renderWithProviders(<ExplorePage />);
    expect(await screen.findByText(/Setup is not applied yet/)).toBeInTheDocument();
    expect(screen.getByText(/checked only against declared recipe evidence/)).toBeInTheDocument();
  });

  it("explains the empty workspace without sample content", async () => {
    exploreRecipes.mockResolvedValue({ planRevision: 0n, profileRevision: 0n, profileConfigured: true, candidates: [] });
    renderWithProviders(<ExplorePage />);
    expect(await screen.findByRole("heading", { name: "No saved meals to explore yet" })).toBeInTheDocument();
    expect(screen.queryByText(/sample|mock/i)).not.toBeInTheDocument();
  });

  it("names the evaluated blockers when saved recipes do not pass eligibility", async () => {
    exploreRecipes.mockResolvedValue({ planRevision: 0n, profileRevision: 2n, profileConfigured: true, savedRecipeCount: 2, candidates: [], blockingReasons: [{ code: "allergen_evidence_unknown", rule: "peanut", reference: "recipe.allergens.peanut", message: "Active allergen restriction lacks decisive evidence." }] });
    renderWithProviders(<ExplorePage />);
    expect(await screen.findByRole("heading", { name: "No saved meals fit yet" })).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Why saved meals were excluded" })).toHaveTextContent("Active allergen restriction lacks decisive evidence. · peanut");
  });

  it("keeps a no-results search visible and offers a reset", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExplorePage />);
    await screen.findByRole("heading", { name: "Lentil soup" });
    await user.type(screen.getByRole("searchbox", { name: "Search eligible saved meals" }), "missing meal");
    expect(await screen.findByText(/No eligible saved meals match/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByRole("heading", { name: "Lentil soup" })).toBeInTheDocument();
  });

  it("distinguishes an offline read from a request failure", async () => {
    const original = navigator.onLine;
    Object.defineProperty(navigator, "onLine", { configurable: true, value: false });
    exploreRecipes.mockRejectedValue(new Error("request failed"));
    renderWithProviders(<ExplorePage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Explore is offline");
    Object.defineProperty(navigator, "onLine", { configurable: true, value: original });
  });

  it("retains a clear request error when the server cannot be read", async () => {
    exploreRecipes.mockRejectedValue(new Error("service unavailable"));
    renderWithProviders(<ExplorePage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("service unavailable");
  });

  it("adds a selected eligible recipe to the requested empty slot", async () => {
    const user = userEvent.setup();
    getPlan.mockResolvedValue({ hasPlan: true, draft: { occurrences: [{ date: "2026-10-09", slotName: "lunch", mode: "fixed", recipeId: "locked-r", recipeRevision: 7, recipeName: "Locked rice", reason: "Owner selected", locked: true }], unresolved: [], inputReferences: ["workspace:w1", "profile:2", "recipe:locked-r:7"], runId: "existing", seed: 0, currentRevision: 4n } });
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner"] });
    await screen.findByRole("heading", { name: "Lentil soup" });
    await user.click(screen.getByRole("button", { name: /Add to Thursday dinner/ }));
    expect(await screen.findByRole("status")).toHaveTextContent("was added to dinner on 2026-10-08");
    expect(getPlan).toHaveBeenCalledWith({ workspaceId: "w1", fromDate: "0001-01-01", toDate: "9999-12-31" });
    expect(applyPlan).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", expectedRevision: 4n, draft: expect.objectContaining({ occurrences: [expect.objectContaining({ date: "2026-10-09", slotName: "lunch", recipeId: "locked-r", recipeRevision: 7, locked: true }), expect.objectContaining({ date: "2026-10-08", slotName: "dinner", recipeId: "r1", recipeRevision: 3, locked: false })] }) }));
  });

  it("requires a preview and explicit confirmation before replacing an occupied slot", async () => {
    const user = userEvent.setup();
    const swapDraft = { occurrences: [{ date: "2026-10-08", slotName: "dinner", recipeId: "r1", recipeRevision: 3, recipeName: "Lentil soup" }], unresolved: [], inputReferences: [], runId: "swap", seed: 0, currentRevision: 4n };
    previewSwap.mockResolvedValue({ revision: 4n, preview: { draft: swapDraft, changes: [{ date: "2026-10-08", slotName: "dinner", beforeName: "Old soup", afterName: "Lentil soup" }], relatedOccurrences: [{ date: "2026-10-09", slotName: "lunch", recipeRevision: 7, recipeName: "Locked rice", locked: true }], preparedBatchImpacts: [], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["r1"], checked: false } }] }, affectedDates: ["2026-10-08"] });
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner&replace=occurrence"] });
    await screen.findByRole("heading", { name: "Lentil soup" });
    await user.click(screen.getByRole("button", { name: "Preview replacing Thursday dinner" }));
    expect(await screen.findByRole("dialog")).toHaveTextContent("Old soup → Lentil soup");
    expect(screen.getByRole("region", { name: "Shopping impact" })).toHaveTextContent("need unknown → unknown");
    expect(screen.getByRole("region", { name: "Related planned meals" })).toHaveTextContent("Locked rice, revision 7");
    expect(applyPlan).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Confirm replacement" }));
    expect(await screen.findByRole("status")).toHaveTextContent("replaced the meal in dinner on 2026-10-08");
    expect(applyPlan).toHaveBeenCalledWith({ workspaceId: "w1", expectedRevision: 4n, draft: swapDraft });
  });

  it("marks changed slot context stale and blocks actions until the latest slot is reviewed", async () => {
    getPlan.mockResolvedValue({ hasPlan: true, draft: { occurrences: [{ date: "2026-10-08", slotName: "dinner", recipeId: "new", recipeRevision: 2, recipeName: "Current meal" }], unresolved: [], inputReferences: [], runId: "current", seed: 0, currentRevision: 5n } });
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner&basePlanRevision=4&expectedRecipeId=old&replace=occurrence"] });
    expect(await screen.findByRole("alert")).toHaveTextContent("This slot changed since you opened Explore");
    expect(screen.getByRole("alert")).toHaveTextContent("Current meal");
    expect(screen.getByRole("button", { name: "Preview replacing Thursday dinner" })).toBeDisabled();
    expect(previewSwap).not.toHaveBeenCalled();
    expect(applyPlan).not.toHaveBeenCalled();
  });

  it("treats a repeated add for the same pinned recipe as already complete", async () => {
    const user = userEvent.setup();
    getPlan.mockResolvedValue({ hasPlan: true, draft: { occurrences: [{ date: "2026-10-08", slotName: "dinner", recipeId: "r1", recipeRevision: 3, recipeName: "Lentil soup" }], unresolved: [], inputReferences: [], runId: "saved", seed: 0, currentRevision: 5n } });
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner"] });
    await screen.findByRole("heading", { name: "Lentil soup" });
    await user.click(screen.getByRole("button", { name: /Add to Thursday dinner/ }));
    expect(await screen.findByRole("status")).toHaveTextContent("is already saved in dinner on 2026-10-08");
    expect(applyPlan).not.toHaveBeenCalled();
  });

  it("fills an intentionally open saved slot without duplicating its occurrence", async () => {
    const user = userEvent.setup();
    getPlan.mockResolvedValue({ hasPlan: true, draft: { occurrences: [{ date: "2026-10-08", slotName: "dinner", mode: "open", recipeId: "", recipeName: "", locked: false }], unresolved: [], inputReferences: [], runId: "open", seed: 0, currentRevision: 5n } });
    renderWithProviders(<ExplorePage />, { routerEntries: ["/meals/explore?date=2026-10-08&slot=dinner"] });
    await screen.findByRole("heading", { name: "Lentil soup" });
    await user.click(screen.getByRole("button", { name: /Add to Thursday dinner/ }));
    await screen.findByText("Lentil soup was added to dinner on 2026-10-08.");
    const [request] = applyPlan.mock.calls[0] ?? [];
    expect(request.draft.occurrences).toHaveLength(1);
    expect(request.draft.occurrences[0]).toMatchObject({ recipeId: "r1", date: "2026-10-08", slotName: "dinner" });
  });
});
