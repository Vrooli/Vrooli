import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  RewritePlanResponseSchema,
  RewriteApplyResponseSchema,
} from "@vrooli/proto-types/typescript-code-graph/v1/graph/graph_pb";
import { OperationStatus } from "@vrooli/proto-types/typescript-code-graph/v1/rewrite/rewrite_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

// The single TypeScriptCodeGraphService client lives in api/graph; api/rewrite
// re-exports it. Mocking api/graph swaps the client everywhere while keeping
// the real operation builders in api/rewrite.
vi.mock("../../api/graph", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/graph")>();
  return {
    ...actual,
    tsCodeGraphClient: {
      extract: vi.fn(),
      rewritePlan: vi.fn(),
      rewriteApply: vi.fn(),
      listFixtures: vi.fn(),
      validateFixture: vi.fn(),
    },
  };
});

import { RewriteTab } from "./RewriteTab";
import { makeFileMoveOp, tsCodeGraphClient } from "../../api/rewrite";

const client = vi.mocked(tsCodeGraphClient);

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

async function addFileMove(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId(selectors.features.rewrite.opsEditor.addFileMove));
  await user.type(screen.getByLabelText(strings.rewrite.ops.fromPath), "old/a.ts");
  await user.type(screen.getByLabelText(strings.rewrite.ops.toPath), "new/a.ts");
}

describe("RewriteTab", () => {
  it("adds a typed operation row", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RewriteTab scenarioPath="/repo" />);
    expect(screen.getByTestId(selectors.features.rewrite.opsEditor.empty)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.features.rewrite.opsEditor.addFileMove));
    expect(screen.getByTestId(selectors.features.rewrite.opRow({ index: 0 }))).toBeInTheDocument();
  });

  it("previews a plan and renders the normalized operations", async () => {
    const user = userEvent.setup();
    client.rewritePlan.mockResolvedValue(
      create(RewritePlanResponseSchema, {
        planId: "plan-abc",
        normalizedOperations: [makeFileMoveOp("old/a.ts", "new/a.ts")],
      }),
    );
    renderWithProviders(<RewriteTab scenarioPath="/repo" />);
    await addFileMove(user);
    await user.click(screen.getByTestId(selectors.features.rewrite.plan.button));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.features.rewrite.plan.result)).toBeInTheDocument();
    });
    expect(client.rewritePlan).toHaveBeenCalledTimes(1);
  });

  it("gates apply behind a confirm dialog", async () => {
    const user = userEvent.setup();
    client.rewritePlan.mockResolvedValue(
      create(RewritePlanResponseSchema, {
        planId: "plan-abc",
        normalizedOperations: [makeFileMoveOp("old/a.ts", "new/a.ts")],
      }),
    );
    client.rewriteApply.mockResolvedValue(
      create(RewriteApplyResponseSchema, {
        planId: "plan-abc",
        dryRun: false,
        results: [{ status: OperationStatus.OK, message: "" }],
      }),
    );
    renderWithProviders(<RewriteTab scenarioPath="/repo" />);
    await addFileMove(user);
    await user.click(screen.getByTestId(selectors.features.rewrite.plan.button));
    await waitFor(() => screen.getByTestId(selectors.features.rewrite.apply.button));

    // Apply does NOT call the API directly — it opens the confirm dialog first.
    await user.click(screen.getByTestId(selectors.features.rewrite.apply.button));
    expect(
      screen.getByTestId(selectors.features.rewrite.apply.confirmDialog.root),
    ).toBeInTheDocument();
    expect(client.rewriteApply).not.toHaveBeenCalled();

    await user.click(screen.getByTestId(selectors.features.rewrite.apply.confirmDialog.confirm));
    await waitFor(() => {
      expect(client.rewriteApply).toHaveBeenCalledTimes(1);
    });
  });
});
