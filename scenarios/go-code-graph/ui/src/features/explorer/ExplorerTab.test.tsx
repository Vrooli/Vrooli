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
    { id: "package:m/a", kind: NodeKind.PACKAGE, name: "a", path: "m/a" },
    { id: "package:m/b", kind: NodeKind.PACKAGE, name: "b", path: "m/b" },
    {
      id: "file:a.go",
      kind: NodeKind.FILE,
      name: "a.go",
      path: "a.go",
      attributes: { package_id: "package:m/a" },
    },
    {
      id: "go_func:a:Run",
      kind: NodeKind.PACKAGE,
      name: "Run",
      path: "a.go",
      attributes: { kind: "go_func", file_id: "file:a.go", exported: "true" },
    },
  ],
  edges: [
    { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: "package:m/a", toNodeId: "package:m/b" },
    { id: "e2", kind: EdgeKind.IMPORT, fromNodeId: "package:m/b", toNodeId: "package:m/a" },
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

  it("surfaces the cycle banner when packages form an import cycle", () => {
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    expect(screen.getByTestId(selectors.features.explorer.cycleBanner)).toBeInTheDocument();
  });

  it("reveals a file's symbols when the file is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    await user.click(screen.getByRole("button", { name: /a\.go/ }));
    expect(
      screen.getByTestId(selectors.features.explorer.drilldown.symbol({ id: "go_func:a:Run" })),
    ).toBeInTheDocument();
  });

  it("filters packages via the filter bar chips", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExplorerTab graph={graph} target="m" />);
    // Filtering to pkg a alone drops the b node from the accessible list.
    await user.click(screen.getByTestId(selectors.features.explorer.filterBar.chip({ key: "m/a" })));
    expect(
      screen.queryByTestId(selectors.features.explorer.accessibleList.item({ id: "package:m/b" })),
    ).not.toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.explorer.accessibleList.item({ id: "package:m/a" })),
    ).toBeInTheDocument();
  });
});
