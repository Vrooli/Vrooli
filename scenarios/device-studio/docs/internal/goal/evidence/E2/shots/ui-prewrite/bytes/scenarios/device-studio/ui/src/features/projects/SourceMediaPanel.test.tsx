import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { projectsClient, uploadSourceMedia, getSourceMediaContent } from "../../api/projects";
import { SourceMediaPanel } from "./SourceMediaPanel";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return {
    ...actual,
    projectsClient: { listProjects: vi.fn(), createProject: vi.fn(), getProject: vi.fn(), renameProject: vi.fn(), listSourceMedia: vi.fn(), getSourceMedia: vi.fn() },
    uploadSourceMedia: vi.fn(),
    getSourceMediaContent: vi.fn(),
  };
});

const record = {
  id: "m1", projectId: "p1", sha256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
  sizeBytes: 3n, mediaType: "image/png", width: 1, height: 1, originalFilename: "capture.png",
  orientation: "identity", colorModel: "RGB", colorProfileStatus: "unspecified", importMethod: "local_upload", decoder: "png",
};

describe("SourceMediaPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [] } as never);
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

  it("keeps the chosen file available for retry after an import failure", async () => {
    const user = userEvent.setup();
    vi.mocked(uploadSourceMedia).mockRejectedValueOnce(new Error("unsupported image profile")).mockRejectedValueOnce(new Error("unsupported image profile"));
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    const file = new File(["abc"], "capture.png", { type: "image/png" });
    await user.upload(screen.getByTestId(selectors.media.fileInput), file);
    await user.click(screen.getByTestId(selectors.media.uploadButton));
    expect(await screen.findByTestId(selectors.media.uploadError)).toHaveTextContent("unsupported image profile");
    expect(screen.getByTestId(selectors.media.fileInput)).toHaveValue("C:\\fakepath\\capture.png");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(uploadSourceMedia).toHaveBeenCalledTimes(2));
    expect(uploadSourceMedia).toHaveBeenLastCalledWith("p1", file);
  });

  it("locks the picker while an import is pending and selects the returned saved record", async () => {
    const user = userEvent.setup();
    let finish!: (value: typeof record) => void;
    vi.mocked(uploadSourceMedia).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }) as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:returned-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    const input = screen.getByTestId(selectors.media.fileInput);
    await user.upload(input, new File(["abc"], "capture.png", { type: "image/png" }));
    await user.click(screen.getByTestId(selectors.media.uploadButton));
    expect(input).toBeDisabled();
    finish(record);
    expect(await screen.findByTestId(selectors.media.image)).toHaveAttribute("alt", "Imported source image: capture.png");
    expect(await screen.findByRole("status")).toHaveTextContent("Imported capture.png and selected it as source media.");
    expect(input).not.toBeDisabled();
    expect(input).toHaveValue("");
    expect(projectsClient.getSourceMedia).toHaveBeenCalledWith({ projectId: "p1", id: record.id });
  });

  it("lists, selects and shows only content whose saved digest matches", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:test-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    expect(await screen.findByTestId(selectors.media.image)).toHaveAttribute("alt", "Imported source image: capture.png");
    await user.click(screen.getByText("Source details"));
    expect(screen.getByText(`SHA-256: ${record.sha256}`)).toBeInTheDocument();
    expect(screen.getByText("Original imported image bytes. This is not a rendered device shot.")).toBeInTheDocument();
  });

  it("withholds the preview when the verified content endpoint rejects the bytes", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia).mockResolvedValue({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockRejectedValue(new Error("source digest verification failed"));
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    expect(await screen.findByRole("alert")).toHaveTextContent("source digest verification failed");
    expect(screen.queryByTestId(selectors.media.image)).not.toBeInTheDocument();
  });

  it("lets a user retry metadata lookup for the selected image", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [record], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMedia)
      .mockRejectedValueOnce(new Error("metadata temporarily unavailable"))
      .mockResolvedValueOnce({ sourceMedia: record } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["abc"], { type: "image/png" }));
    const testURL = class extends URL { static createObjectURL = vi.fn(() => "blob:retried-source"); static revokeObjectURL = vi.fn(); };
    vi.stubGlobal("URL", testURL);
    renderWithProviders(<SourceMediaPanel projectId="p1" />);
    await user.click(await screen.findByTestId(selectors.media.selectButton({ id: "m1" })));
    const failure = await screen.findByRole("alert");
    expect(failure).toHaveTextContent("metadata temporarily unavailable");
    await user.click(within(failure).getByRole("button", { name: "Retry" }));
    expect(await screen.findByTestId(selectors.media.image)).toBeInTheDocument();
  });
});
