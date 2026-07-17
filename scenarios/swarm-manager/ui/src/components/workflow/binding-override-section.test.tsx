import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { WorkflowBindingsPanel } from "./binding-override-section";
import { compatibleModesForOperation } from "../../lib/agent-ops-utils";
import { agentOperationsService } from "../../services";
import { selectors } from "../../consts/selectors";
import type {
  AgentOpsTarget,
  WorkflowBindingContribution,
  WorkflowBindingOverrideDocument,
  WorkflowCompatibleMode,
  WorkflowOperationBinding,
  WorkflowResolvedBinding,
} from "../../types/agent-operations";

vi.mock("../../services", () => ({
  agentOperationsService: {
    getResolvedBindings: vi.fn(),
    listCompatibleModes: vi.fn(),
    listBindingOverrides: vi.fn(),
    putBindingOverride: vi.fn(),
    deleteBindingOverride: vi.fn(),
  },
}));

const initiativeTarget: AgentOpsTarget = { kind: "initiative", id: "init-a" };
const itemTarget: AgentOpsTarget = { kind: "backlog-item", id: "execute/item-1" };

function binding(overrides: Partial<WorkflowOperationBinding> = {}): WorkflowOperationBinding {
  return {
    operation: "backlog.execute",
    operationVersion: "1.0.0",
    layer: "system-default",
    ownerKind: "system",
    ownerId: "",
    mode: "backlog-fixup",
    modeRevision: "rev-3",
    disabled: false,
    ...overrides,
  };
}

function resolvedRow(overrides: Partial<WorkflowResolvedBinding> = {}): WorkflowResolvedBinding {
  const winning = overrides.binding ?? binding();
  return {
    operation: winning?.operation ?? "backlog.execute",
    operationVersion: winning?.operationVersion ?? "1.0.0",
    resolved: true,
    binding: winning,
    policyId: "policy-1",
    policyRevision: "p1",
    error: "",
    errorMessage: "",
    contributions: winning ? [{ binding: winning, winning: true }] : [],
    ...overrides,
  };
}

function compatibleMode(overrides: Partial<WorkflowCompatibleMode> = {}): WorkflowCompatibleMode {
  return {
    mode: "backlog-fixup",
    modeRevision: "rev-3",
    modeDigest: "sha256:abcdef1234567890",
    targetKind: "backlog-item",
    verdicts: [
      { operation: "backlog.execute", operationVersion: "1.0.0", compatible: true, reason: "" },
    ],
    ...overrides,
  };
}

function overrideDoc(
  overrides: Partial<WorkflowBindingOverrideDocument> = {},
): WorkflowBindingOverrideDocument {
  return {
    binding: binding({
      layer: "initiative-override",
      ownerKind: "initiative",
      ownerId: "init-a",
      mode: "backlog-research",
      modeRevision: "rev-1",
    }),
    file: "backlog.execute@1.0.0.json",
    revision: "f2",
    updatedAt: "2026-07-01T00:00:00Z",
    ...overrides,
  };
}

function renderPanel(target: AgentOpsTarget = initiativeTarget) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <WorkflowBindingsPanel target={target} />
    </QueryClientProvider>,
  );
  return { queryClient, ...utils };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([resolvedRow()]);
  vi.mocked(agentOperationsService.listCompatibleModes).mockResolvedValue([compatibleMode()]);
  vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([]);
});

describe("WorkflowBindingsPanel", () => {
  it("renders each resolvable operation with winning mode, exact revision, and source label", async () => {
    renderPanel();
    const row = await screen.findByTestId(selectors.workflowBindings.row);
    expect(row).toHaveTextContent("backlog.execute");
    expect(row).toHaveTextContent("@1.0.0");
    expect(row).toHaveTextContent("backlog-fixup");
    expect(row).toHaveTextContent("rev rev-3");
    expect(within(row).getByTestId(selectors.workflowBindings.rowSource)).toHaveTextContent(
      "System default",
    );
  });

  it("notes snapshot semantics: changes apply to operations started after the change", async () => {
    renderPanel();
    expect(
      await screen.findByText(/changes apply to operations started after this change/i),
    ).toBeInTheDocument();
  });

  it("renders unresolved operations honestly with the server's typed error", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([
      resolvedRow({
        resolved: false,
        binding: null,
        contributions: [],
        error: "no-binding",
        errorMessage: "no binding at any layer",
      }),
    ]);
    renderPanel();
    const error = await screen.findByTestId(selectors.workflowBindings.rowError);
    expect(error).toHaveTextContent("no-binding");
    expect(error).toHaveTextContent("no binding at any layer");
  });

  it("makes the layer ladder visible when an item override shadows an initiative override", async () => {
    const systemBinding = binding();
    const initiativeBinding = binding({
      layer: "initiative-override",
      ownerKind: "initiative",
      ownerId: "init-a",
      mode: "backlog-research",
      modeRevision: "rev-1",
    });
    const itemBinding = binding({
      layer: "backlog-item-override",
      ownerKind: "backlog-item",
      ownerId: "execute/item-1",
      mode: "backlog-workshop",
      modeRevision: "rev-9",
    });
    const contributions: WorkflowBindingContribution[] = [
      { binding: itemBinding, winning: true },
      { binding: systemBinding, winning: false },
      { binding: initiativeBinding, winning: false },
    ];
    vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([
      resolvedRow({ binding: itemBinding, contributions }),
    ]);
    renderPanel(itemTarget);

    const chips = await screen.findAllByTestId(selectors.workflowBindings.layerChip);
    expect(chips).toHaveLength(3);
    // Precedence order preserved in display: system < initiative < item.
    expect(chips[0]).toHaveTextContent("System default");
    expect(chips[1]).toHaveTextContent("Initiative override");
    expect(chips[2]).toHaveTextContent("Item override");
    // The item layer wins; the shadowed layers say so.
    expect(chips[2]).toHaveAttribute("data-winning", "true");
    expect(chips[1]).not.toHaveAttribute("data-winning");
    expect(chips[1]).toHaveTextContent("shadowed");
  });

  describe("override dialog", () => {
    it("offers ONLY server-compatible modes for the operation", async () => {
      vi.mocked(agentOperationsService.listCompatibleModes).mockResolvedValue([
        compatibleMode({ mode: "backlog-fixup", modeRevision: "rev-3" }),
        compatibleMode({
          mode: "backlog-research",
          modeRevision: "rev-5",
          verdicts: [
            {
              operation: "backlog.execute",
              operationVersion: "1.0.0",
              compatible: false,
              reason: "missing capability",
            },
          ],
        }),
      ]);
      renderPanel();
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.workflowBindings.overrideButton));

      const options = await screen.findAllByTestId(
        selectors.workflowBindings.overrideDialogModeOption,
      );
      expect(options).toHaveLength(1);
      expect(options[0]).toHaveTextContent("backlog-fixup");
      // Compatibility preview shows revision + digest prefix.
      expect(options[0]).toHaveTextContent("rev rev-3");
      expect(options[0]).toHaveTextContent("sha256:abcdef123456");
      expect(screen.queryByText("backlog-research")).toBeNull();
    });

    it("puts an override pinned to the operation version and invalidates binding queries", async () => {
      vi.mocked(agentOperationsService.putBindingOverride).mockResolvedValue({
        stored: binding(),
        file: "backlog.execute@1.0.0.json",
        revision: "f3",
      });
      const { queryClient } = renderPanel();
      const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.workflowBindings.overrideButton));
      await screen.findAllByTestId(selectors.workflowBindings.overrideDialogModeOption);
      await user.click(screen.getByTestId(selectors.workflowBindings.overrideDialogConfirm));

      await waitFor(() => {
        expect(agentOperationsService.putBindingOverride).toHaveBeenCalledWith({
          owner: initiativeTarget,
          operation: "backlog.execute",
          operationVersion: "1.0.0",
          mode: "backlog-fixup",
          modeRevision: "rev-3",
        });
      });
      await waitFor(() => {
        expect(invalidateSpy).toHaveBeenCalledWith({
          queryKey: ["agent-ops", "resolved-bindings"],
        });
        expect(invalidateSpy).toHaveBeenCalledWith({
          queryKey: ["agent-ops", "binding-overrides"],
        });
      });
    });
  });

  describe("override provenance + reset to inherited", () => {
    it("shows provenance for an override stored at this owner's layer", async () => {
      vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
      renderPanel();
      const provenance = await screen.findByTestId(
        selectors.workflowBindings.overrideProvenance,
      );
      expect(provenance).toHaveTextContent("initiative init-a");
      expect(provenance).toHaveTextContent("rev f2");
      expect(provenance).toHaveTextContent(/updated/);
    });

    it("resets to inherited through a confirm dialog calling deleteBindingOverride", async () => {
      vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
      vi.mocked(agentOperationsService.deleteBindingOverride).mockResolvedValue({ found: true });
      renderPanel();
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.workflowBindings.resetButton));

      // Destructive action is confirm-gated.
      const dialog = await screen.findByTestId(selectors.workflowBindings.resetConfirmDialog);
      expect(dialog).toHaveTextContent(/inherited/i);
      expect(agentOperationsService.deleteBindingOverride).not.toHaveBeenCalled();

      await user.click(screen.getByTestId(selectors.workflowBindings.resetConfirmButton));
      await waitFor(() => {
        expect(agentOperationsService.deleteBindingOverride).toHaveBeenCalledWith(
          initiativeTarget,
          "backlog.execute",
          "1.0.0",
        );
      });
    });

    it("does not offer reset when no override exists at this owner's layer", async () => {
      renderPanel();
      await screen.findByTestId(selectors.workflowBindings.row);
      expect(screen.queryByTestId(selectors.workflowBindings.resetButton)).toBeNull();
    });
  });

  describe("stale-revision honesty", () => {
    it("flags an override pinned to a revision that no longer matches the mode's current revision", async () => {
      // Override pins backlog-research@rev-1; the catalog's current revision is rev-5.
      vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
      vi.mocked(agentOperationsService.listCompatibleModes).mockResolvedValue([
        compatibleMode({ mode: "backlog-research", modeRevision: "rev-5" }),
      ]);
      renderPanel();
      const stale = await screen.findByTestId(selectors.workflowBindings.staleRevision);
      expect(stale).toHaveTextContent(/pinned to older revision/i);
      expect(stale).toHaveTextContent("rev-1");
      expect(stale).toHaveTextContent("rev-5");
    });

    it("shows no stale indicator when the pinned revision equals the current one", async () => {
      vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([
        overrideDoc({
          binding: binding({
            layer: "initiative-override",
            ownerKind: "initiative",
            ownerId: "init-a",
            mode: "backlog-research",
            modeRevision: "rev-5",
          }),
        }),
      ]);
      vi.mocked(agentOperationsService.listCompatibleModes).mockResolvedValue([
        compatibleMode({ mode: "backlog-research", modeRevision: "rev-5" }),
      ]);
      renderPanel();
      await screen.findByTestId(selectors.workflowBindings.overrideProvenance);
      expect(screen.queryByTestId(selectors.workflowBindings.staleRevision)).toBeNull();
    });
  });
});

describe("accessibility", () => {
  it("exposes the row actions as buttons with accessible names", async () => {
    vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
    renderPanel();
    await screen.findByTestId(selectors.workflowBindings.row);
    expect(screen.getByRole("button", { name: /override/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reset to inherited/i })).toBeInTheDocument();
  });

  it("opens an accessible override dialog: role=dialog labelled by its title, focus inside, Escape closes", async () => {
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));

    const dialog = await screen.findByRole("dialog", {
      name: /override binding — backlog\.execute/i,
    });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    // Focus lands inside the dialog on open.
    await waitFor(() => expect(dialog).toHaveFocus());

    await user.keyboard("{Escape}");
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: /override binding — backlog\.execute/i }),
      ).not.toBeInTheDocument(),
    );
  });

  it("exposes compatible modes as a labelled radiogroup with checked state", async () => {
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));

    const group = await screen.findByRole("radiogroup", { name: "Compatible modes" });
    const option = within(group).getByRole("radio", { name: /backlog-fixup/i });
    expect(option).toHaveAttribute("aria-checked", "true");
  });

  it("presents the reset confirmation as an alertdialog labelled by its title", async () => {
    vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /reset to inherited/i }));
    expect(
      await screen.findByRole("alertdialog", { name: "Reset to inherited binding" }),
    ).toBeInTheDocument();
  });
});

describe("error states", () => {
  it("renders the query error affordance when GetResolvedBindings fails", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockRejectedValue(
      new Error("policy store unavailable"),
    );
    renderPanel();
    expect(await screen.findByText("policy store unavailable")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.workflowBindings.row)).toBeNull();
  });

  it("surfaces the compatible-modes query error inside the override dialog", async () => {
    vi.mocked(agentOperationsService.listCompatibleModes).mockRejectedValue(
      new Error("mode catalog unreadable"),
    );
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));

    const dialog = await screen.findByRole("dialog", {
      name: /override binding — backlog\.execute/i,
    });
    expect(within(dialog).getByText("mode catalog unreadable")).toBeInTheDocument();
    // No modes can be offered, so no submit is possible.
    expect(
      screen.getByTestId(selectors.workflowBindings.overrideDialogConfirm),
    ).toBeDisabled();
  });

  it("keeps the dialog open and shows the server's fail-closed message when the put fails", async () => {
    vi.mocked(agentOperationsService.putBindingOverride).mockRejectedValue(
      new Error("incompatible-mode: backlog-fixup does not implement backlog.execute@1.0.0"),
    );
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));
    await screen.findAllByTestId(selectors.workflowBindings.overrideDialogModeOption);
    await user.click(screen.getByTestId(selectors.workflowBindings.overrideDialogConfirm));

    expect(
      await screen.findByText(
        "incompatible-mode: backlog-fixup does not implement backlog.execute@1.0.0",
      ),
    ).toBeInTheDocument();
    // Fail-closed: nothing was applied, the dialog stays open for repair.
    expect(
      screen.getByRole("dialog", { name: /override binding — backlog\.execute/i }),
    ).toBeInTheDocument();
  });

  it("keeps the reset confirm open and shows the typed message when the delete fails", async () => {
    vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
    vi.mocked(agentOperationsService.deleteBindingOverride).mockRejectedValue(
      new Error("override-not-found: no override stored at this layer"),
    );
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /reset to inherited/i }));
    await user.click(screen.getByTestId(selectors.workflowBindings.resetConfirmButton));

    expect(
      await screen.findByText("override-not-found: no override stored at this layer"),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workflowBindings.resetConfirmDialog)).toBeInTheDocument();
  });
});

describe("empty states", () => {
  it("says so when no operations are resolvable for the scope", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([]);
    renderPanel();
    expect(
      await screen.findByText(/no operations are resolvable for this scope/i),
    ).toBeInTheDocument();
  });

  it("communicates an empty compatible-mode catalog and offers no enabled submit", async () => {
    vi.mocked(agentOperationsService.listCompatibleModes).mockResolvedValue([]);
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));

    expect(
      await screen.findByTestId(selectors.workflowBindings.overrideDialogEmpty),
    ).toHaveTextContent(/no compatible modes/i);
    expect(
      screen.queryByTestId(selectors.workflowBindings.overrideDialogModeOption),
    ).toBeNull();
    expect(
      screen.getByTestId(selectors.workflowBindings.overrideDialogConfirm),
    ).toBeDisabled();
  });
});

describe("loading states", () => {
  it("renders a loading affordance while resolved bindings are pending", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockReturnValue(
      new Promise(() => {}),
    );
    renderPanel();
    const status = await screen.findByRole("status");
    expect(status).toHaveTextContent(/loading resolved bindings/i);
  });

  it("renders a loading affordance for compatible modes inside the dialog", async () => {
    vi.mocked(agentOperationsService.listCompatibleModes).mockReturnValue(
      new Promise(() => {}),
    );
    renderPanel();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /override/i }));

    const dialog = await screen.findByRole("dialog", {
      name: /override binding — backlog\.execute/i,
    });
    expect(within(dialog).getByRole("status")).toHaveTextContent(/loading compatible modes/i);
  });
});

describe("incompatible / deleted overrides surfaced by resolution", () => {
  it("renders the typed incompatible-mode error and keeps repair affordances reachable", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([
      resolvedRow({
        resolved: false,
        binding: null,
        contributions: [],
        error: "incompatible-mode",
        errorMessage: "backlog-research@rev-1 does not implement backlog.execute@1.0.0",
      }),
    ]);
    vi.mocked(agentOperationsService.listBindingOverrides).mockResolvedValue([overrideDoc()]);
    renderPanel();

    const error = await screen.findByTestId(selectors.workflowBindings.rowError);
    expect(error).toHaveTextContent("incompatible-mode");
    expect(error).toHaveTextContent(
      "backlog-research@rev-1 does not implement backlog.execute@1.0.0",
    );
    // Repair paths stay reachable: pick a compatible mode or reset the override.
    expect(screen.getByRole("button", { name: /override/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /reset to inherited/i })).toBeEnabled();
  });

  it("renders the typed deleted-revision error honestly", async () => {
    vi.mocked(agentOperationsService.getResolvedBindings).mockResolvedValue([
      resolvedRow({
        resolved: false,
        binding: null,
        contributions: [],
        error: "deleted-revision",
        errorMessage: "pinned revision rev-1 no longer exists",
      }),
    ]);
    renderPanel();

    const error = await screen.findByTestId(selectors.workflowBindings.rowError);
    expect(error).toHaveTextContent("deleted-revision");
    expect(error).toHaveTextContent("pinned revision rev-1 no longer exists");
    expect(screen.getByRole("button", { name: /override/i })).toBeEnabled();
  });
});

describe("compatibleModesForOperation (display filter over server verdicts)", () => {
  it("matches version-agnostic verdicts to any pinned version", () => {
    const modes = [
      compatibleMode({
        verdicts: [
          { operation: "backlog.execute", operationVersion: "", compatible: true, reason: "" },
        ],
      }),
    ];
    expect(compatibleModesForOperation(modes, "backlog.execute", "2.0.0")).toHaveLength(1);
  });

  it("excludes verdicts pinned to a different version", () => {
    const modes = [
      compatibleMode({
        verdicts: [
          { operation: "backlog.execute", operationVersion: "1.0.0", compatible: true, reason: "" },
        ],
      }),
    ];
    expect(compatibleModesForOperation(modes, "backlog.execute", "2.0.0")).toHaveLength(0);
  });

  it("never offers incompatible modes", () => {
    const modes = [
      compatibleMode({
        verdicts: [
          { operation: "backlog.execute", operationVersion: "1.0.0", compatible: false, reason: "x" },
        ],
      }),
    ];
    expect(compatibleModesForOperation(modes, "backlog.execute", "1.0.0")).toHaveLength(0);
  });
});
