import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn(),
    extractGraph: vi.fn(),
  },
}));

import { graphClient } from "../../api/graph";
import { ApiError } from "../../api/client";
import { ErrorEnvelopeSchema } from "@vrooli/proto-types/architecture-cartographer/v1/errors/errors_pb";
import { create } from "@bufbuild/protobuf";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ActiveSnapshotsPanel } from "./ActiveSnapshotsPanel";

afterEach(() => {
  cleanup();
  vi.mocked(graphClient.listGraphSnapshots).mockReset();
});

describe("ActiveSnapshotsPanel", () => {
  it("renders the empty state when the API returns no snapshots", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockResolvedValue({
      snapshots: [],
      nextPageToken: "",
    } as unknown as Awaited<ReturnType<typeof graphClient.listGraphSnapshots>>);
    renderWithProviders(<ActiveSnapshotsPanel />);

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.targets.activeSnapshots.empty),
      ).toBeInTheDocument(),
    );
  });

  it("renders a row per snapshot with an Open link", async () => {
    vi.mocked(graphClient.listGraphSnapshots).mockResolvedValue({
      snapshots: [
        {
          id: "snap:demo:1",
          scenario: "demo",
          contentHash: "h",
          languages: [],
          extractedAt: undefined,
          extractionMs: BigInt(10),
          files: [],
          packages: [],
          symbols: [],
          imports: [],
        },
      ],
      nextPageToken: "",
    } as unknown as Awaited<ReturnType<typeof graphClient.listGraphSnapshots>>);

    renderWithProviders(<ActiveSnapshotsPanel />);
    await waitFor(() =>
      expect(
        screen.getByTestId(
          selectors.features.targets.activeSnapshots.item({ id: "snap:demo:1" }),
        ),
      ).toBeInTheDocument(),
    );
  });

  it("renders the error state with a retry affordance when the API rejects", async () => {
    const envelope = create(ErrorEnvelopeSchema, {
      code: "unavailable",
      message: "boom",
    });
    vi.mocked(graphClient.listGraphSnapshots).mockRejectedValue(new ApiError(envelope, 503));
    renderWithProviders(<ActiveSnapshotsPanel />);

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.targets.activeSnapshots.error),
      ).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.features.targets.activeSnapshots.error),
    ).toHaveTextContent("boom");
  });
});
