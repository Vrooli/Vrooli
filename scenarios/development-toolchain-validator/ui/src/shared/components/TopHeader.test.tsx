import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";

import {
  GoldenSchema,
  ListGoldensResponseSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";
import {
  GetGoldenSummaryResponseSchema,
  GoldenSummarySchema,
  TupleVerdictSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/report/report_pb";
import {
  ListStaleResponseSchema,
  StaleEntrySchema,
  StaleKind,
} from "@vrooli/proto-types/development-toolchain-validator/v1/staleness/staleness_pb";
import { ResponseSchema } from "@vrooli/proto-types/development-toolchain-validator/v1/health/health_pb";
import {
  TupleKind,
  Verdict,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_record/validation_record_pb";

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { TopHeader } from "./TopHeader";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return {
    ...actual,
    fetchHealth: vi.fn(),
  };
});

vi.mock("../../api/golden", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/golden")>();
  return {
    ...actual,
    goldenClient: { listGoldens: vi.fn() },
  };
});

vi.mock("../../api/report", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/report")>();
  return {
    ...actual,
    reportClient: { getGoldenSummary: vi.fn() },
  };
});

vi.mock("../../api/staleness", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/staleness")>();
  return {
    ...actual,
    stalenessClient: { listStale: vi.fn() },
  };
});

import { fetchHealth } from "../../api/health";
import { goldenClient } from "../../api/golden";
import { reportClient } from "../../api/report";
import { stalenessClient } from "../../api/staleness";

describe("TopHeader", () => {
  it("shows converged, no-stale, healthy status", async () => {
    vi.mocked(fetchHealth).mockResolvedValue(create(ResponseSchema, { status: "healthy" }));
    vi.mocked(goldenClient.listGoldens).mockResolvedValue(
      create(ListGoldensResponseSchema, {
        goldens: [create(GoldenSchema, { slug: "reference-react-vite" })],
      }),
    );
    vi.mocked(reportClient.getGoldenSummary).mockResolvedValue(
      create(GetGoldenSummaryResponseSchema, {
        summary: create(GoldenSummarySchema, {
          goldenSlug: "reference-react-vite",
          skillVerdicts: [
            create(TupleVerdictSchema, {
              tupleKind: TupleKind.SKILL,
              subjectId: "progress",
              latestVerdict: Verdict.PASS,
            }),
          ],
        }),
      }),
    );
    vi.mocked(stalenessClient.listStale).mockResolvedValue(
      create(ListStaleResponseSchema, { entries: [] }),
    );

    renderWithProviders(<TopHeader />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.nav.topHeaderHealth)).toHaveTextContent("healthy");
    });
    expect(screen.getByTestId(selectors.nav.topHeaderConvergence)).toHaveAttribute(
      "data-variant",
      "verdict-pass",
    );
    expect(screen.getByTestId(selectors.nav.topHeaderStale)).toHaveAttribute(
      "data-variant",
      "neutral",
    );
  });

  it("shows stale and offline degraded states with a menu toggle", async () => {
    const onMenuToggle = vi.fn();
    vi.mocked(fetchHealth).mockRejectedValue(new Error("offline"));
    vi.mocked(goldenClient.listGoldens).mockResolvedValue(
      create(ListGoldensResponseSchema, { goldens: [] }),
    );
    vi.mocked(stalenessClient.listStale).mockResolvedValue(
      create(ListStaleResponseSchema, {
        entries: [
          create(StaleEntrySchema, {
            skillId: "progress",
            goldenSlug: "reference-react-vite",
            kind: StaleKind.SKILL_VERSION,
          }),
        ],
      }),
    );

    renderWithProviders(<TopHeader onMenuToggle={onMenuToggle} />);

    screen.getByTestId(selectors.nav.topHeaderMenu).click();
    expect(onMenuToggle).toHaveBeenCalled();
    await waitFor(() => {
      expect(screen.getByTestId(selectors.nav.topHeaderHealth)).toHaveTextContent("offline");
    });
    expect(screen.getByTestId(selectors.nav.topHeaderStale)).toHaveAttribute(
      "data-variant",
      "verdict-stale",
    );
  });
});
