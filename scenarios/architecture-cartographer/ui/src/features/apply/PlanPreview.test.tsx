import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { OperationKind } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { PlanPreview } from "./PlanPreview";

afterEach(() => cleanup());

describe("PlanPreview", () => {
  it("renders empty states for absent plans and plans without operations", () => {
    const { rerender } = renderWithProviders(<PlanPreview plan={undefined} />);
    expect(screen.getByTestId(selectors.features.apply.plan.empty)).toBeInTheDocument();

    rerender(<PlanPreview plan={{ id: "p-1", operations: [] } as never} />);
    expect(screen.getByTestId(selectors.features.apply.plan.empty)).toBeInTheDocument();
  });

  it("renders all operation kind label branches", () => {
    renderWithProviders(
      <PlanPreview
        plan={
          {
            id: "p-1",
            operations: [
              OperationKind.MOVE_FILE,
              OperationKind.REWRITE_IMPORT,
              OperationKind.DELETE_FILE,
              OperationKind.CREATE_FILE,
              OperationKind.UNSPECIFIED,
            ].map((kind, index) => ({
              id: `op-${index}`,
              kind,
              fromPath: index % 2 === 0 ? `from-${index}.go` : "",
              toPath: index % 2 === 1 ? `to-${index}.go` : "",
            })),
          } as never
        }
      />,
    );

    const root = screen.getByTestId(selectors.features.apply.plan.root);
    expect(root).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(6);
    expect(screen.getByTestId(selectors.shared.dataTable.row({ id: "op-0" }))).toHaveTextContent(
      "from-0.go",
    );
    expect(screen.getByTestId(selectors.shared.dataTable.row({ id: "op-1" }))).toHaveTextContent(
      "to-1.go",
    );
    expect(root).toHaveTextContent("—");
  });
});
