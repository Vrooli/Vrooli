import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { RetentionTab } from "./RetentionTab";
import { parseHumanBytes, bytesToHumanReadable } from "./byteFormat";

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...actual,
    fetchRetentionConfig: vi.fn(),
    updateRetentionConfig: vi.fn(),
  };
});

const apiModule = await import("../../lib/api");
const fetchMock = apiModule.fetchRetentionConfig as unknown as ReturnType<typeof vi.fn>;
const updateMock = apiModule.updateRetentionConfig as unknown as ReturnType<typeof vi.fn>;

function renderWithClient(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

describe("parseHumanBytes", () => {
  it.each([
    ["10", 10],
    ["1024", 1024],
    ["10 GiB", 10 * 1024 ** 3],
    ["10gib", 10 * 1024 ** 3],
    ["500 MB", 500_000_000],
    ["500MB", 500_000_000],
    ["0", 0],
    ["1.5 GiB", Math.round(1.5 * 1024 ** 3)],
  ])("parses %s → %d", (input, expected) => {
    expect(parseHumanBytes(input)).toBe(expected);
  });

  it.each(["", "abc", "-1", "-1 GiB", "10 furlongs", "10..5 GiB"])("rejects %s", (input) => {
    expect(parseHumanBytes(input)).toBeNull();
  });
});

describe("bytesToHumanReadable", () => {
  it.each([
    [0, "0"],
    [512, "512 B"],
    [1024, "1 KiB"],
    [1024 ** 2, "1 MiB"],
    [10 * 1024 ** 3, "10 GiB"],
    [1.5 * 1024 ** 3, "1.5 GiB"],
  ])("formats %d → %s", (input, expected) => {
    expect(bytesToHumanReadable(input)).toBe(expected);
  });
});

describe("RetentionTab", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    updateMock.mockReset();
  });

  it("loads current config and renders defaults", async () => {
    fetchMock.mockResolvedValueOnce({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());
    expect(screen.getByTestId<HTMLInputElement>("retention-age-days").value).toBe("90");
    expect(screen.getByTestId<HTMLInputElement>("retention-size").value).toBe("10 GiB");
    expect(screen.getByTestId<HTMLInputElement>("retention-per-project").value).toBe("0");
    expect(screen.getByTestId<HTMLButtonElement>("retention-save")).toBeDisabled();
  });

  it("only sends changed fields on save (partial PUT)", async () => {
    fetchMock.mockResolvedValue({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    updateMock.mockResolvedValue({
      maxArchiveAgeDays: 60,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());

    const ageInput = screen.getByTestId<HTMLInputElement>("retention-age-days");
    fireEvent.change(ageInput, { target: { value: "60" } });

    const save = screen.getByTestId<HTMLButtonElement>("retention-save");
    expect(save).not.toBeDisabled();

    await act(async () => {
      fireEvent.click(save);
    });

    expect(updateMock).toHaveBeenCalledTimes(1);
    expect(updateMock).toHaveBeenCalledWith({ maxArchiveAgeDays: 60 });
  });

  it("rejects negative age and disables save", async () => {
    fetchMock.mockResolvedValueOnce({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 0,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());

    fireEvent.change(screen.getByTestId("retention-age-days"), { target: { value: "-5" } });
    expect(screen.getByTestId("retention-age-days-error")).toBeInTheDocument();
    expect(screen.getByTestId<HTMLButtonElement>("retention-save")).toBeDisabled();
  });

  it("rejects garbage size input", async () => {
    fetchMock.mockResolvedValueOnce({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 0,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());

    fireEvent.change(screen.getByTestId("retention-size"), { target: { value: "10 furlongs" } });
    expect(screen.getByTestId("retention-size-error")).toBeInTheDocument();
    expect(screen.getByTestId<HTMLButtonElement>("retention-save")).toBeDisabled();
  });

  it("Reset reverts unsaved edits to last loaded config", async () => {
    fetchMock.mockResolvedValueOnce({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());

    const ageInput = screen.getByTestId<HTMLInputElement>("retention-age-days");
    fireEvent.change(ageInput, { target: { value: "30" } });
    expect(ageInput.value).toBe("30");

    fireEvent.click(screen.getByTestId("retention-reset"));
    expect(ageInput.value).toBe("90");
    expect(screen.getByTestId<HTMLButtonElement>("retention-save")).toBeDisabled();
  });

  it("explicit zero submits as zero (disable lever)", async () => {
    fetchMock.mockResolvedValue({
      maxArchiveAgeDays: 90,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    updateMock.mockResolvedValue({
      maxArchiveAgeDays: 0,
      maxArchiveSizeBytes: 10 * 1024 ** 3,
      maxArchivesPerProject: 0,
    });
    renderWithClient(<RetentionTab />);
    await waitFor(() => expect(screen.queryByTestId("retention-tab")).toBeInTheDocument());

    fireEvent.change(screen.getByTestId("retention-age-days"), { target: { value: "0" } });
    await act(async () => {
      fireEvent.click(screen.getByTestId("retention-save"));
    });
    expect(updateMock).toHaveBeenCalledWith({ maxArchiveAgeDays: 0 });
  });
});
