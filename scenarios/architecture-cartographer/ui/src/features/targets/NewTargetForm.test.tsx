import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn(),
  },
}));

import { graphClient } from "../../api/graph";
import { ApiError } from "../../api/client";
import { ErrorEnvelopeSchema } from "@vrooli/proto-types/architecture-cartographer/v1/errors/errors_pb";
import { create } from "@bufbuild/protobuf";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { NewTargetForm } from "./NewTargetForm";

const makeStorage = (initial: Record<string, string> = {}) => {
  const store: Record<string, string> = { ...initial };
  return {
    getItem: (key: string) => (key in store ? store[key]! : null),
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    snapshot: () => ({ ...store }),
  };
};

afterEach(() => {
  cleanup();
  vi.mocked(graphClient.extractGraph).mockReset();
});

describe("NewTargetForm", () => {
  it("rejects an empty scenario with a validation error", async () => {
    const user = userEvent.setup();
    renderWithProviders(<NewTargetForm recentTargetsStorage={makeStorage()} />);

    await user.click(screen.getByTestId(selectors.features.targets.newForm.submitButton));

    expect(
      screen.getByTestId(selectors.features.targets.newForm.scenarioInputError),
    ).toBeInTheDocument();
    expect(graphClient.extractGraph).not.toHaveBeenCalled();
  });

  it("rejects scenario names containing slashes or whitespace", async () => {
    const user = userEvent.setup();
    renderWithProviders(<NewTargetForm recentTargetsStorage={makeStorage()} />);

    await user.type(
      screen.getByTestId(selectors.features.targets.newForm.scenarioInput),
      "bad name/",
    );
    await user.click(screen.getByTestId(selectors.features.targets.newForm.submitButton));

    expect(
      screen.getByTestId(selectors.features.targets.newForm.scenarioInputError),
    ).toBeInTheDocument();
    expect(graphClient.extractGraph).not.toHaveBeenCalled();
  });

  it("calls extractGraph and shows success + open-workspace link on success", async () => {
    vi.mocked(graphClient.extractGraph).mockResolvedValue({
      snapshot: {
        id: "snap:abc",
        scenario: "demo",
        contentHash: "h",
        languages: [],
        extractedAt: undefined,
        extractionMs: BigInt(42),
        files: [],
        packages: [],
        symbols: [],
        imports: [],
      } as unknown as Awaited<ReturnType<typeof graphClient.extractGraph>>["snapshot"],
      fromCache: false,
    } as Awaited<ReturnType<typeof graphClient.extractGraph>>);

    const user = userEvent.setup();
    const storage = makeStorage();
    renderWithProviders(
      <NewTargetForm recentTargetsStorage={storage} now={() => new Date("2026-01-01T00:00:00Z")} />,
    );

    await user.type(
      screen.getByTestId(selectors.features.targets.newForm.scenarioInput),
      "demo",
    );
    await user.click(screen.getByTestId(selectors.features.targets.newForm.submitButton));

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.targets.newForm.successBanner),
      ).toBeInTheDocument(),
    );

    expect(graphClient.extractGraph).toHaveBeenCalledWith({
      scenario: "demo",
      languages: [],
      idempotencyKey: "",
    });

    const openLink = screen.getByTestId(
      selectors.features.targets.newForm.openWorkspaceLink,
    );
    expect(openLink).toHaveAttribute("href", "/targets/demo");

    // The form should have appended the scenario to the recent-targets list
    // via the injected storage.
    const persisted = JSON.parse(storage.snapshot()["cartographer.recentTargets"] ?? "[]");
    expect(persisted).toHaveLength(1);
    expect(persisted[0].scenario).toBe("demo");
  });

  it("renders the API error message when extractGraph rejects with an ApiError", async () => {
    const envelope = create(ErrorEnvelopeSchema, {
      code: "failed_precondition",
      message: "scenario not found",
    });
    vi.mocked(graphClient.extractGraph).mockRejectedValue(new ApiError(envelope, 412));

    const user = userEvent.setup();
    renderWithProviders(<NewTargetForm recentTargetsStorage={makeStorage()} />);

    await user.type(
      screen.getByTestId(selectors.features.targets.newForm.scenarioInput),
      "missing",
    );
    await user.click(screen.getByTestId(selectors.features.targets.newForm.submitButton));

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.features.targets.newForm.errorBanner),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.features.targets.newForm.errorBanner),
    ).toHaveTextContent("scenario not found");
  });
});
