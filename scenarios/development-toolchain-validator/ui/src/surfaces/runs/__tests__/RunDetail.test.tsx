/**
 * RunDetail surface — status + verdict render, terminal vs pending, not-found.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";

import { create } from "@bufbuild/protobuf";
import {
  ValidationRunSchema,
  GetResponseSchema,
  Status,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_run/validation_run_pb";
import {
  TupleKind,
  Verdict,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_record/validation_record_pb";

import { ROUTE_PATTERNS } from "../../../routes.generated";
import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";

vi.mock("../../../api/validationRun", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../../api/validationRun")>();
  return {
    ...actual,
    validationRunClient: { start: vi.fn(), get: vi.fn(), listActive: vi.fn() },
  };
});

import { RunDetail } from "../RunDetail";
import { validationRunClient } from "../../../api/validationRun";

const renderDetail = (id: string) =>
  renderWithProviders(
    <Routes>
      <Route path={ROUTE_PATTERNS.runDetail} element={<RunDetail />} />
    </Routes>,
    { routerEntries: [`/runs/${id}`] },
  );

const makeGetResponse = (status: Status, verdict: Verdict) =>
  create(GetResponseSchema, {
    run: create(ValidationRunSchema, {
      id: "run-1",
      subjectId: "progress",
      goldenSlug: "reference-react-vite",
      tupleKind: TupleKind.SKILL,
      status,
      terminalVerdict: verdict,
      agentManagerRunId: "am-run-1",
    }),
  });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RunDetail", () => {
  it("renders the terminal verdict when the run is terminal", async () => {
    vi.mocked(validationRunClient.get).mockResolvedValue(
      makeGetResponse(Status.TERMINAL, Verdict.PASS),
    );
    renderDetail("run-1");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.detailStatus)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.runs.detailVerdict)).toBeInTheDocument();
  });

  it("renders the not-found empty state when the run is absent", async () => {
    vi.mocked(validationRunClient.get).mockResolvedValue(
      create(GetResponseSchema, {}),
    );
    renderDetail("missing");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.empty)).toBeInTheDocument();
    });
  });

  it("renders an error when the get query fails", async () => {
    vi.mocked(validationRunClient.get).mockRejectedValue(new Error("boom"));
    renderDetail("run-1");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.runs.error)).toBeInTheDocument();
    });
  });
});
