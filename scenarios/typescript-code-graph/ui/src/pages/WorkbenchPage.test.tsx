import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { ExtractResponseSchema } from "@vrooli/proto-types/typescript-code-graph/v1/graph/graph_pb";
import { CodeGraphSchema, NodeKind, EdgeKind } from "@vrooli/proto-types/common/v1/code_graph_pb";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/graph", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/graph")>();
  return {
    ...actual,
    tsCodeGraphClient: {
      extract: vi.fn(),
      rewritePlan: vi.fn(),
      rewriteApply: vi.fn(),
      listFixtures: vi.fn().mockResolvedValue({ fixtures: [] }),
      validateFixture: vi.fn(),
    },
  };
});

import { WorkbenchPage } from "./WorkbenchPage";
import { tsCodeGraphClient } from "../api/graph";

const client = vi.mocked(tsCodeGraphClient);

const extractResponse = create(ExtractResponseSchema, {
  graph: create(CodeGraphSchema, {
    nodes: [
      { id: "ts_module:src/a.ts", kind: NodeKind.MODULE, name: "a.ts", path: "src/a.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
      { id: "ts_module:src/b.ts", kind: NodeKind.MODULE, name: "b.ts", path: "src/b.ts", attributes: { kind: "TS_NODE_KIND_MODULE" } },
      { id: "file:src/a.ts", kind: NodeKind.FILE, name: "a.ts", path: "src/a.ts", attributes: { language: "typescript" } },
    ],
    edges: [{ id: "e", kind: EdgeKind.IMPORT, fromNodeId: "ts_module:src/a.ts", toNodeId: "ts_module:src/b.ts" }],
  }),
  warnings: [],
  extractionMs: 42n,
  graphHash: "deadbeefcafef00d",
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkbenchPage", () => {
  it("renders the idle empty state before any extraction", () => {
    renderWithProviders(<WorkbenchPage />);
    expect(screen.getByTestId(selectors.pages.workbench)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workbench.status.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workbench.extractBar.root)).toBeInTheDocument();
  });

  it("extracts on submit and renders the stats header + tabs", async () => {
    const user = userEvent.setup();
    client.extract.mockResolvedValue(extractResponse);
    renderWithProviders(<WorkbenchPage />);

    await user.type(screen.getByTestId(selectors.workbench.extractBar.target), "scenarios/typescript-code-graph");
    await user.click(screen.getByTestId(selectors.workbench.extractBar.submit));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workbench.stats.root)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workbench.stats.modules)).toHaveTextContent("2");
    expect(screen.getByTestId(selectors.workbench.stats.files)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.workbench.stats.imports)).toHaveTextContent("1");
    // Graph tab is the default; its canvas should be visible.
    expect(screen.getByTestId(selectors.features.explorer.canvas.root)).toBeInTheDocument();
    expect(client.extract).toHaveBeenCalledWith({
      scenarioPath: "scenarios/typescript-code-graph",
    });
  });

  it("switches to the rewrite tab", async () => {
    const user = userEvent.setup();
    client.extract.mockResolvedValue(extractResponse);
    renderWithProviders(<WorkbenchPage />);
    await user.type(screen.getByTestId(selectors.workbench.extractBar.target), "x");
    await user.click(screen.getByTestId(selectors.workbench.extractBar.submit));
    await waitFor(() => screen.getByTestId(selectors.workbench.stats.root));

    await user.click(screen.getByTestId(selectors.ui.tabs.trigger({ value: "rewrite" })));
    expect(screen.getByTestId(selectors.features.rewrite.root)).toBeInTheDocument();
  });
});
