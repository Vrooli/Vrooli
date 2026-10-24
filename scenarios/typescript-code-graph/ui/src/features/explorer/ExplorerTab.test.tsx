import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { CodeGraphSchema, NodeKind, EdgeKind } from "@vrooli/proto-types/common/v1/code_graph_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ExplorerTab } from "./ExplorerTab";

const graph = create(CodeGraphSchema, {
  nodes: [
    { id: "ts_module:src/a.ts", kind: NodeKind.MODULE, name: "a.ts", path: "src/a.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
    { id: "ts_module:src/b.ts", kind: NodeKind.MODULE, name: "b.ts", path: "src/b.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
    {
      id: "file:src/a.ts",
      kind: NodeKind.FILE,
      name: "a.ts",
      path: "src/a.ts",
      attributes: { language: "typescript" },
    },
    {
      id: "ts_function:src/a.ts:run",
      kind: NodeKind.UNSPECIFIED,
      name: "run",
      path: "src/a.ts",
      attributes: { kind: "TS_NODE_KIND_FUNCTION", exported: "true" },
    },
  ],
  edges: [
    { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/a.ts", toNodeId: "ts_module:src/b.ts" },
    { id: "e2", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/b.ts", toNodeId: "ts_module:src/a.ts" },
  ],
});

describe("ExplorerTab", () => {
  afterEach(cleanup);

  it("renders the canvas, legend, accessible list, and file drill-down", () => {
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.legend.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.accessibleList.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.explorer.drilldown.root)).toBeInTheDocument();
  });

  it("surfaces the cycle banner when modules form an import cycle", () => {
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.cycleBanner)).toBeInTheDocument();
  });

  it("reveals a file's symbols when the file is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    await user.click(screen.getByRole("button", { name: /src\/a\.ts/ }));
    expect(
      screen.getByTestId(selectors.features.explorer.drilldown.symbol({ id: "ts_function:src/a.ts:run" })),
    ).toBeInTheDocument();
  });

  it("filters modules via the filter bar chips", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    // Filtering to module a alone drops the b node from the accessible list.
    await user.click(screen.getByTestId(selectors.features.explorer.filterBar.chip({ key: "src/a.ts" })));
    expect(
      screen.queryByTestId(selectors.features.explorer.accessibleList.item({ id: "ts_module:src/b.ts" })),
    ).not.toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.accessibleList.item({ id: "ts_module:src/a.ts" })),
    ).toBeInTheDocument();
  });
});
