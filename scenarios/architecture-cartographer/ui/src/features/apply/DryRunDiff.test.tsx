import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { DryRunDiff } from "./DryRunDiff";
import { makePlan, makeOperation } from "./flow/fixtures";
import { OperationKind } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

afterEach(() => cleanup());

describe("DryRunDiff", () => {
  it("renders an empty state when there is no plan", () => {
    renderWithProviders(<DryRunDiff plan={undefined} />);
    expect(screen.getByTestId(selectors.features.apply.dryRun.empty)).toBeInTheDocument();
  });

  it("renders the diff view when the plan has operations", () => {
    const plan = makePlan({
      operations: [
        makeOperation({ kind: OperationKind.MOVE_FILE, fromPath: "a", toPath: "b" }),
      ],
    });
    renderWithProviders(<DryRunDiff plan={plan} />);
    expect(screen.getByTestId(selectors.features.apply.dryRun.root)).toBeInTheDocument();
  });
});
