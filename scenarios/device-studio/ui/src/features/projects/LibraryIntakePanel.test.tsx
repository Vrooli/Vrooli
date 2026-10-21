import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { IntakeCandidateRevisionSchema, IntakeInspectionSchema } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { importLocalGeometryCandidate, listLocalGeometryCandidates } from "../../api/library-intake";
import { LibraryIntakePanel } from "./LibraryIntakePanel";

vi.mock("../../api/library-intake", () => ({
  MAX_LOCAL_GEOMETRY_BYTES: 128 * 1024 * 1024,
  listLocalGeometryCandidates: vi.fn(),
  importLocalGeometryCandidate: vi.fn(),
  downloadLocalGeometryCandidate: vi.fn(),
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
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

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
});
