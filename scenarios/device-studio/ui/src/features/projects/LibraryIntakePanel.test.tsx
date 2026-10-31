import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { IntakeCandidateRevisionSchema, IntakeInspectionSchema } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { correctLocalGeometryMaterial, downloadLocalGeometryCandidate, getLocalGeometryMaterials, getLocalGeometryPreview, importLocalGeometryCandidate, listLocalGeometryCandidates } from "../../api/library-intake";
import { importPreparedPhotograph } from "../../api/physical";
import { LibraryIntakePanel } from "./LibraryIntakePanel";

vi.mock("../../api/library-intake", () => ({
  MAX_LOCAL_GEOMETRY_BYTES: 128 * 1024 * 1024,
  listLocalGeometryCandidates: vi.fn(),
  importLocalGeometryCandidate: vi.fn(),
  downloadLocalGeometryCandidate: vi.fn(),
  getLocalGeometryMaterials: vi.fn(),
  getLocalGeometryPreview: vi.fn(),
  correctLocalGeometryMaterial: vi.fn(),
}));

vi.mock("../../api/physical", () => ({ importPreparedPhotograph: vi.fn() }));

const candidate = create(IntakeCandidateRevisionSchema, {
  id: "intake-test", revision: 1, revisionSha256: "a".repeat(64), displayName: "Phone model",
  originalFilename: "phone.glb", originalSha256: "b".repeat(64), sizeBytes: 256n,
  reviewStatus: "unreviewed", renderStatus: "not_reviewed",
  inspection: create(IntakeInspectionSchema, { profile: "test-v1", profileVersion: 1, supported: false, finding: "unsupported_by_current_geometry_inspector" }),
});

describe("LibraryIntakePanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [], nextAfterCandidateId: "" } as never);
    vi.mocked(importLocalGeometryCandidate).mockResolvedValue(candidate);
    vi.mocked(getLocalGeometryMaterials).mockResolvedValue({ candidate_id: candidate.id, revision: candidate.revision, revision_sha256: candidate.revisionSha256, glb_sha256: candidate.originalSha256, materials: [{ index: 0, name: "case", base_color_factor: [1, 1, 1, 1], metallic_factor: 0, roughness_factor: 0.8, emissive_factor: [0, 0, 0], editable: true }] });
    vi.mocked(getLocalGeometryPreview).mockResolvedValue({ blob: new Blob(["bounded-png"]), preview_sha256: "c".repeat(64), vertex_colors: true, width: 768, height: 256 });
    vi.mocked(correctLocalGeometryMaterial).mockResolvedValue({ id: "intake-child", revision: 1, revision_sha256: "c".repeat(64), original_sha256: "d".repeat(64), review_status: "unreviewed", render_status: "not_reviewed", parent_candidate_id: candidate.id, parent_revision: candidate.revision, parent_sha256: candidate.revisionSha256, import_method: "local_material_correction", material_correction_lineage: { material_name: "case", before: { roughness_factor: 0.8 }, after: { roughness_factor: 0.2 } } });
    vi.mocked(importPreparedPhotograph).mockResolvedValue({
      revision: { sceneId: "photographic-background-test", revision: 1, state: "candidate", candidate: { sceneId: "photographic-background-test" } },
      libraryBundleId: "photographic-background-test", libraryRevision: 1, libraryManifestSha256: "e".repeat(64),
      scaleStatus: "estimated_monocular_scale_not_measured", estimatedScaleM: 1.4,
    } as never);
  });
  afterEach(() => { cleanup(); localStorage.clear(); vi.restoreAllMocks(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("imports local bytes as an unreviewed candidate and refreshes the catalog", async () => {
    vi.mocked(listLocalGeometryCandidates)
      .mockResolvedValueOnce({ candidates: [], nextAfterCandidateId: "" } as never)
      .mockResolvedValueOnce({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    const file = new File([new Uint8Array([1, 2, 3])], "phone.glb", { type: "model/gltf-binary" });
    await user.upload(screen.getByLabelText("Local GLB file"), file);
    await user.type(screen.getByLabelText("Display name"), "Phone model");
    await user.type(screen.getAllByLabelText("Source description")[1]!, "Downloaded from a local archive");
    await user.type(screen.getAllByLabelText("License claim")[1]!, "CC-BY-4.0");
    await user.type(screen.getAllByLabelText("Attribution claim")[1]!, "Original author credit");
    await user.click(screen.getByText("Technical details"));
    await user.clear(screen.getByLabelText("Candidate ID"));
    await user.type(screen.getByLabelText("Candidate ID"), "reviewed-source-copy");
    await user.click(screen.getByText("Technical details"));
    await user.click(screen.getByRole("button", { name: "Import candidate" }));
    expect(await screen.findByText(/Review: unreviewed · rendering: not_reviewed/)).toBeInTheDocument();
    expect(await screen.findByText(/Not supported by the current geometry inspector/)).toBeInTheDocument();
    const fileInput = screen.getByLabelText("Local GLB file") as HTMLInputElement;
    expect(fileInput.value).toBe("");
    const technicalDetails = screen.getByText("Technical details").closest("details");
    expect(technicalDetails).not.toHaveAttribute("open");
    await user.click(screen.getByText("Technical details"));
    expect(screen.getByLabelText("Candidate ID")).toBeInTheDocument();
    expect(importLocalGeometryCandidate).toHaveBeenCalledWith(file, expect.objectContaining({
      candidateId: "reviewed-source-copy", displayName: "Phone model", sourceLabel: "Downloaded from a local archive",
      licenseClaim: "CC-BY-4.0", attribution: "Original author credit",
    }));
    expect(listLocalGeometryCandidates).toHaveBeenCalledTimes(2);
    expect(screen.getByText(/Import does not grant rights, qualify geometry, or make this candidate selectable for rendering/)).toBeInTheDocument();
    await user.upload(fileInput, file);
    await user.type(screen.getByLabelText("Display name"), "Phone model again");
    await user.type(screen.getAllByLabelText("Source description")[1]!, "Downloaded from a local archive");
    await user.click(screen.getByRole("button", { name: "Import candidate" }));
    await waitFor(() => expect(importLocalGeometryCandidate).toHaveBeenCalledTimes(2));
  });

  it("imports the exact reviewed photograph through the typed scene-owner RPC and shows immutable revisions", async () => {
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    const importPanel = within(screen.getByTestId("prepared-photo-import"));
    const preparationID = "a".repeat(64);
    await user.type(importPanel.getByLabelText("Preparation ID / bundle SHA-256"), preparationID);
    await user.type(importPanel.getByLabelText("Source description"), "Generated empty desk plate");
    await user.type(importPanel.getByLabelText("License claim"), "CC-BY-4.0");
    await user.type(importPanel.getByLabelText("Attribution claim"), "Generated for scene preparation");
    await user.click(importPanel.getByLabelText("Allow local rendering under the declared source use"));
    await user.type(importPanel.getByLabelText("Source-use rationale"), "The source license permits local compositing and render evaluation.");
    expect(importPanel.getByText(/declared edit and optional local-render permissions are recorded separately/)).toBeInTheDocument();
    expect(importPanel.queryByText(/Library assets limited to edit only/)).not.toBeInTheDocument();
    await user.click(importPanel.getByRole("button", { name: "Import reviewed preparation" }));
    expect(await screen.findByText("Created scene candidate photographic-background-test revision 1.")).toBeInTheDocument();
    expect(screen.getByText(/Library bundle photographic-background-test revision 1/)).toBeInTheDocument();
    expect(screen.getByText(/estimated scale 1.4 m/)).toBeInTheDocument();
    expect(importPreparedPhotograph).toHaveBeenCalledWith({
      preparationId: preparationID, expectedBundleSha256: preparationID, licenseId: "CC-BY-4.0",
      attribution: "Generated for scene preparation", sourceLabel: "Generated empty desk plate",
      allowLocalRender: true, sourceUseRationale: "The source license permits local compositing and render evaluation.",
    });
  });

  it("loads the next immutable candidate page when requested", async () => {
    const next = create(IntakeCandidateRevisionSchema, { ...candidate, id: "intake-next", displayName: "Next geometry" });
    vi.mocked(listLocalGeometryCandidates).mockImplementation(async (after) => after
      ? { candidates: [next], nextAfterCandidateId: "" } as never
      : { candidates: [candidate], nextAfterCandidateId: "intake-test" } as never);
    renderWithProviders(<LibraryIntakePanel />);
    expect(await screen.findByText("Phone model")).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Load more candidates" }));
    expect(await screen.findByText("Next geometry")).toBeInTheDocument();
    expect(listLocalGeometryCandidates).toHaveBeenLastCalledWith("intake-test");
  });

  it("shows a digest-bound vertex-color preview while keeping the candidate unreviewed", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    vi.stubGlobal("URL", { createObjectURL: vi.fn(() => "blob:preview"), revokeObjectURL: vi.fn() });
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Preview (unreviewed)" }));
    expect(await screen.findByTestId(`intake-preview-${candidate.id}`)).toBeInTheDocument();
    expect(screen.getByText(/embedded vertex colors/)).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Three-view preview of an unreviewed imported geometry candidate" })).toBeInTheDocument();
    expect(getLocalGeometryPreview).toHaveBeenCalledWith(candidate);
    expect(screen.getByText(/Review: unreviewed · rendering: not_reviewed/)).toBeInTheDocument();
  });

  it("does not make a candidate appear if import fails", async () => {
    vi.mocked(importLocalGeometryCandidate).mockRejectedValueOnce(new Error("GLB exceeds the local geometry size limit"));
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.upload(screen.getByLabelText("Local GLB file"), new File(["bytes"], "bad.glb"));
    await user.type(screen.getByLabelText("Display name"), "Bad model");
    await user.type(screen.getAllByLabelText("Source description")[1]!, "local file");
    await user.click(screen.getByRole("button", { name: "Import candidate" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("GLB exceeds the local geometry size limit");
    expect(screen.queryByText(/Review: unreviewed/)).not.toBeInTheDocument();
  });

  it("creates a deterministic local material child and retries the exact request after remount", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    vi.mocked(correctLocalGeometryMaterial).mockRejectedValueOnce(new Error("response interrupted"));
    const user = userEvent.setup();
    const first = renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    await user.type(await screen.findByLabelText("New roughness · leave blank to keep"), "0.2");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    await screen.findByText("An exact correction request is saved locally for safe retry. Its fields stay locked until it succeeds or you discard it.");
    const firstRequest = vi.mocked(correctLocalGeometryMaterial).mock.calls.at(0)?.[0];
    if (!firstRequest) throw new Error("first correction request was not dispatched");
    expect(firstRequest).toMatchObject({ parent_candidate_id: candidate.id, parent_revision: candidate.revision, parent_revision_sha256: candidate.revisionSha256, material_index: 0, material_name: "case", roughness_factor: 0.2 });
    first.unmount();
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const second = renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    await screen.findByText("An exact correction request is saved locally for safe retry. Its fields stay locked until it succeeds or you discard it.");
    await user.click(screen.getByRole("button", { name: "Retry same material edit" }));
    await waitFor(() => expect(correctLocalGeometryMaterial).toHaveBeenCalledTimes(2));
    expect(vi.mocked(correctLocalGeometryMaterial).mock.calls.at(1)?.[0]).toEqual(firstRequest);
    expect(await screen.findByText("Material edit lineage")).toBeInTheDocument();
    expect(screen.getByText(/Child GLB SHA-256 ddddd/)).toBeInTheDocument();
    second.unmount();
  });

  it("fails closed when the browser cannot persist a correction retry identity", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage denied"); });
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    await user.type(await screen.findByLabelText("New roughness · leave blank to keep"), "0.2");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    expect(await screen.findByText("This browser could not save the exact retry request. No correction was sent.")).toBeInTheDocument();
    expect(correctLocalGeometryMaterial).not.toHaveBeenCalled();
  });

  it("applies edits to the selected material and preserves the exact factors in the child request", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    vi.mocked(getLocalGeometryMaterials).mockResolvedValue({
      candidate_id: candidate.id, revision: candidate.revision, revision_sha256: candidate.revisionSha256,
      glb_sha256: candidate.originalSha256, materials: [
        { index: 0, name: "case", base_color_factor: [1, 1, 1, 1], metallic_factor: 0, roughness_factor: 0.8, emissive_factor: [0, 0, 0], editable: true },
        { index: 1, name: "display trim", base_color_factor: [0.5, 0.5, 0.5, 1], metallic_factor: 0.1, roughness_factor: 0.6, emissive_factor: [0, 0, 0], editable: true },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    await user.selectOptions(screen.getByRole("combobox"), "1");
    await user.type(screen.getByLabelText("New metallic · leave blank to keep"), "0.3");
    await user.type(screen.getByLabelText("New base color RGBA · comma-separated, leave blank to keep"), "0.2,0.3,0.4,1");
    await user.type(screen.getByLabelText("New emissive RGB · comma-separated, leave blank to keep"), "0.1,0.2,0.3");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    await waitFor(() => expect(correctLocalGeometryMaterial).toHaveBeenCalledTimes(1));
    expect(correctLocalGeometryMaterial).toHaveBeenCalledWith(expect.objectContaining({
      parent_candidate_id: candidate.id, material_index: 1, material_name: "display trim",
      metallic_factor: 0.3, base_color_factor: [0.2, 0.3, 0.4, 1], emissive_factor: [0.1, 0.2, 0.3],
    }));
  });

  it("lets the user discard a persisted correction retry without sending it", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const key = `device-studio-material-correction:${candidate.id}@${candidate.revision}:${candidate.revisionSha256}`;
    localStorage.setItem(key, JSON.stringify({
      parent_candidate_id: candidate.id, parent_revision: candidate.revision,
      parent_revision_sha256: candidate.revisionSha256, child_candidate_id: "intake-pending-child",
      material_index: 0, material_name: "case", roughness_factor: 0.2,
    }));
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    expect(await screen.findByText("An exact correction request is saved locally for safe retry. Its fields stay locked until it succeeds or you discard it.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Discard pending edit and start a new one" }));
    expect(screen.queryByText("An exact correction request is saved locally for safe retry. Its fields stay locked until it succeeds or you discard it.")).not.toBeInTheDocument();
    expect(localStorage.getItem(key)).toBeNull();
    expect(correctLocalGeometryMaterial).not.toHaveBeenCalled();
  });

  it("downloads exact imported geometry and reports a failed download", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const createObjectURL = vi.fn(() => "blob:download");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL });
    vi.mocked(downloadLocalGeometryCandidate).mockResolvedValue(new Blob(["verified geometry"]));
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await screen.findByText("Phone model");
    await user.click(screen.getByRole("button", { name: "Download verified original" }));
    await waitFor(() => expect(createObjectURL).toHaveBeenCalledOnce());
    expect(click).toHaveBeenCalledOnce();
    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob));

    vi.mocked(downloadLocalGeometryCandidate).mockRejectedValueOnce(new Error("download digest mismatch"));
    await user.click(screen.getByRole("button", { name: "Download verified original" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("download digest mismatch");
  });

  it("rejects empty components in base-color and emissive tuples", async () => {
    vi.mocked(listLocalGeometryCandidates).mockResolvedValue({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.click(await screen.findByRole("button", { name: "Edit material" }));
    const baseColor = screen.getByLabelText("New base color RGBA · comma-separated, leave blank to keep");
    await user.type(baseColor, "0.2,,0.8,1");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Enter finite factor values between 0 and 1");
    expect(correctLocalGeometryMaterial).not.toHaveBeenCalled();
    await user.clear(baseColor);
    await user.type(baseColor, "0.2,NaN,0.8,1");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    expect(correctLocalGeometryMaterial).not.toHaveBeenCalled();
    await user.clear(baseColor);
    await user.type(screen.getByLabelText("New emissive RGB · comma-separated, leave blank to keep"), "0.1, ,0.4");
    await user.click(screen.getByRole("button", { name: "Create unreviewed corrected child" }));
    expect(correctLocalGeometryMaterial).not.toHaveBeenCalled();
  });
});
