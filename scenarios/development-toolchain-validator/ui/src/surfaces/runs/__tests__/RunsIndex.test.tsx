/**
 * RunsIndex surface — loading / empty / active-rows states + start-run form.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { create } from "@bufbuild/protobuf";
import {
  ValidationRunSchema,
  ListActiveResponseSchema,
  StartResponseSchema,
  Status,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_run/validation_run_pb";
import { TupleKind } from "@vrooli/proto-types/development-toolchain-validator/v1/validation_record/validation_record_pb";

import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";

vi.mock("../../../api/validationRun", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../api/validationRun")>();
  return {
    ...actual,
    validationRunClient: {
      start: vi.fn(),
      get: vi.fn(),
      listActive: vi.fn(),
    },
  };
});

import { RunsIndex } from "../RunsIndex";
import { validationRunClient } from "../../../api/validationRun";

const makeRun = (id: string, status: Status = Status.RUNNING) =>
  create(ValidationRunSchema, {
    id,
    subjectId: "progress",
    goldenSlug: "reference-react-vite",
    tupleKind: TupleKind.SKILL,
    status,
  });

const emptyActive = () => create(ListActiveResponseSchema, { runs: [] });

beforeEach(() => {
  vi.mocked(validationRunClient.listActive).mockResolvedValue(emptyActive());
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RunsIndex", () => {
  it("renders the start-run form", async () => {
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.startForm)).toBeInTheDocument();
    });
  });

  it("renders the empty state when there are no active runs", async () => {
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.empty)).toBeInTheDocument();
    });
  });

  it("renders one row per active run", async () => {
    vi.mocked(validationRunClient.listActive).mockResolvedValue(
      create(ListActiveResponseSchema, {
        runs: [makeRun("run-1"), makeRun("run-2")],
      }),
    );
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.runs.row).length).toBe(2);
    });
  });

  it("starts a run when the form is submitted", async () => {
    const user = userEvent.setup();
    vi.mocked(validationRunClient.start).mockResolvedValue(
      create(StartResponseSchema, { run: makeRun("run-new", Status.QUEUED) }),
    );
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.startSubject)).toBeInTheDocument();
    });
    await user.type(screen.getByTestId(selectors.runs.startSubject), "progress");
    await user.type(
      screen.getByTestId(selectors.runs.startGolden),
      "reference-react-vite",
    );
    await user.click(screen.getByTestId(selectors.runs.startSubmit));
    await waitFor(() => {
      expect(validationRunClient.start).toHaveBeenCalledWith(
        expect.objectContaining({
          subjectId: "progress",
          goldenSlug: "reference-react-vite",
          tupleKind: TupleKind.SKILL,
        }),
      );
    });
  });

  it("starts a forced tool run when requested", async () => {
    const user = userEvent.setup();
    vi.mocked(validationRunClient.start).mockResolvedValue(
      create(StartResponseSchema, { run: makeRun("run-new", Status.QUEUED) }),
    );
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.startKind)).toBeInTheDocument();
    });
    await user.selectOptions(screen.getByTestId(selectors.runs.startKind), "tool");
    await user.type(screen.getByTestId(selectors.runs.startSubject), "test-genie");
    await user.type(
      screen.getByTestId(selectors.runs.startGolden),
      "reference-react-vite",
    );
    await user.click(screen.getByTestId(selectors.runs.startForce));
    await user.click(screen.getByTestId(selectors.runs.startSubmit));
    await waitFor(() => {
      expect(validationRunClient.start).toHaveBeenCalledWith(
        expect.objectContaining({
          subjectId: "test-genie",
          tupleKind: TupleKind.TOOL,
          force: true,
        }),
      );
    });
  });

  it("renders an error when listActive fails", async () => {
    vi.mocked(validationRunClient.listActive).mockRejectedValue(
      new Error("boom"),
    );
    renderWithProviders(<RunsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.error)).toBeInTheDocument();
    });
  });
});
