import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  DiffOp,
  makeDiffCell,
  makeDiffRow,
  makeDiffVersionsResponse,
  makeListVersionsResponse,
  makeVersion,
} from "./mocks/factories";
import { makeVersionsMocks } from "./mocks/versions";

vi.mock("../../api/versions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/versions")>();
  return { ...actual, ...makeVersionsMocks() };
});

import { VersionsCard } from "./VersionsCard";

describe("VersionsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders empty state when no versions are recorded", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(makeListVersionsResponse());
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.versions.list)).not.toBeInTheDocument();
  });

  it("renders a row per version with version/sha/recorded labels", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v1", version: "1.0.0", contentSha256: "aaa111bbb222ccc" }),
          makeVersion({ id: "v2", version: "1.0.1", contentSha256: "ddd333eee444fff" }),
        ],
      }),
    );
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.list)).toBeInTheDocument();
    });
    const items = screen.getAllByTestId(selectors.versions.item);
    expect(items).toHaveLength(2);
    expect(items[0]?.textContent ?? "").toContain("v1.0.0");
    expect(items[1]?.textContent ?? "").toContain("v1.0.1");
    const shas = screen.getAllByTestId(selectors.versions.itemSha).map((n) => n.textContent);
    expect(shas[0] ?? "").toContain("aaa111bbb222");
    expect(shas[1] ?? "").toContain("ddd333eee444");
  });

  it("forwards from/to into diffVersions and renders the response rows", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({
        versions: [
          makeVersion({ id: "v1", version: "1.0.0" }),
          makeVersion({ id: "v2", version: "1.0.1" }),
        ],
      }),
    );
    vi.mocked(versionsClient.diffVersions).mockResolvedValueOnce(
      makeDiffVersionsResponse({
        fromLabel: "1.0.0",
        toLabel: "1.0.1",
        additions: 1,
        removals: 1,
        rows: [
          makeDiffRow(
            { lineNumber: 1, text: "alpha", op: DiffOp.EQUAL },
            { lineNumber: 1, text: "alpha", op: DiffOp.EQUAL },
          ),
          makeDiffRow(
            { lineNumber: 2, text: "beta", op: DiffOp.REMOVE },
            makeDiffCell({ op: DiffOp.EMPTY }),
          ),
          makeDiffRow(
            makeDiffCell({ op: DiffOp.EMPTY }),
            { lineNumber: 2, text: "BETA", op: DiffOp.ADD },
          ),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    // Wait for the listVersions response to land so the diff selects
    // have populated options beyond the "—" sentinel.
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.versions.item)).toHaveLength(2);
    });
    await user.selectOptions(
      screen.getByTestId(selectors.versions.diff.fromSelect),
      "1.0.0",
    );
    await user.selectOptions(
      screen.getByTestId(selectors.versions.diff.toSelect),
      "1.0.1",
    );
    await user.click(screen.getByTestId(selectors.versions.diff.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.diff.summary)).toBeInTheDocument();
    });
    expect(vi.mocked(versionsClient.diffVersions)).toHaveBeenCalledWith({
      componentId: "cmp-1",
      from: "1.0.0",
      to: "1.0.1",
    });
    const summary = screen.getByTestId(selectors.versions.diff.summary).textContent;
    expect(summary).toContain("+1");
    expect(summary).toContain("-1");
    expect(screen.getAllByTestId(selectors.versions.diff.row)).toHaveLength(3);
    // Spot-check that add/remove cells render with their text + marker.
    const table = screen.getByTestId(selectors.versions.diff.table);
    expect(table.textContent).toContain("alpha");
    expect(table.textContent).toContain("beta");
    expect(table.textContent).toContain("BETA");
  });

  it("run button is disabled until both sides are chosen", async () => {
    const { versionsClient } = await import("../../api/versions");
    vi.mocked(versionsClient.listVersions).mockResolvedValueOnce(
      makeListVersionsResponse({ versions: [makeVersion({ version: "1.0.0" })] }),
    );
    renderWithProviders(<VersionsCard componentId="cmp-1" />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.versions.diff.runButton)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.versions.diff.runButton)).toBeDisabled();
  });
});
