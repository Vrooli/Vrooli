import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { IntakeCandidateRevisionSchema, IntakeInspectionSchema } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { correctLocalGeometryMaterial, getLocalGeometryMaterials, importLocalGeometryCandidate, listLocalGeometryCandidates } from "../../api/library-intake";
import { LibraryIntakePanel } from "./LibraryIntakePanel";

vi.mock("../../api/library-intake", () => ({
  MAX_LOCAL_GEOMETRY_BYTES: 128 * 1024 * 1024,
  listLocalGeometryCandidates: vi.fn(),
  importLocalGeometryCandidate: vi.fn(),
  downloadLocalGeometryCandidate: vi.fn(),
  getLocalGeometryMaterials: vi.fn(),
  correctLocalGeometryMaterial: vi.fn(),
}));

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
    vi.mocked(correctLocalGeometryMaterial).mockResolvedValue({ id: "intake-child", revision: 1, revision_sha256: "c".repeat(64), original_sha256: "d".repeat(64), review_status: "unreviewed", render_status: "not_reviewed", parent_candidate_id: candidate.id, parent_revision: candidate.revision, parent_sha256: candidate.revisionSha256, import_method: "local_material_correction", material_correction_lineage: { material_name: "case", before: { roughness_factor: 0.8 }, after: { roughness_factor: 0.2 } } });
  });
  afterEach(() => { cleanup(); localStorage.clear(); vi.restoreAllMocks(); vi.clearAllMocks(); });

  it("imports local bytes as an unreviewed candidate and refreshes the catalog", async () => {
    vi.mocked(listLocalGeometryCandidates)
      .mockResolvedValueOnce({ candidates: [], nextAfterCandidateId: "" } as never)
      .mockResolvedValueOnce({ candidates: [candidate], nextAfterCandidateId: "" } as never);
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    const file = new File([new Uint8Array([1, 2, 3])], "phone.glb", { type: "model/gltf-binary" });
    await user.upload(screen.getByLabelText("Local GLB file"), file);
    await user.type(screen.getByLabelText("Display name"), "Phone model");
    await user.type(screen.getByLabelText("Source description"), "Downloaded from a local archive");
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
      candidateId: expect.stringMatching(/^intake-/), displayName: "Phone model", sourceLabel: "Downloaded from a local archive",
    }));
    expect(listLocalGeometryCandidates).toHaveBeenCalledTimes(2);
    expect(screen.getByText(/Import does not grant rights, qualify geometry, or make this candidate selectable for rendering/)).toBeInTheDocument();
    await user.upload(fileInput, file);
    await user.type(screen.getByLabelText("Display name"), "Phone model again");
    await user.type(screen.getByLabelText("Source description"), "Downloaded from a local archive");
    await user.click(screen.getByRole("button", { name: "Import candidate" }));
    await waitFor(() => expect(importLocalGeometryCandidate).toHaveBeenCalledTimes(2));
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

  it("does not make a candidate appear if import fails", async () => {
    vi.mocked(importLocalGeometryCandidate).mockRejectedValueOnce(new Error("GLB exceeds the local geometry size limit"));
    const user = userEvent.setup();
    renderWithProviders(<LibraryIntakePanel />);
    await user.upload(screen.getByLabelText("Local GLB file"), new File(["bytes"], "bad.glb"));
    await user.type(screen.getByLabelText("Display name"), "Bad model");
    await user.type(screen.getByLabelText("Source description"), "local file");
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
