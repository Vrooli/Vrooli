import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import type { DesignDocument } from "../design-types";
import { DEFAULT_TEMPLATE } from "../templates/library";
import type { TemplateNode } from "../templates/spec";
import {
  DesignProposalError,
  type DesignProposal,
  type ProposalTransport,
  type StudioProposal,
} from "./proposal";
import { StudioAssist, type AssistScope } from "./StudioAssist";

const headline: TemplateNode = {
  id: "page.headline",
  type: "text",
  props: { text: "Current headline", variant: "body" },
};

function design(id = "design-a", overrides: Partial<DesignDocument> = {}): DesignDocument {
  const page: TemplateNode = {
    id: "page",
    type: "page",
    props: {},
    children: [headline],
  };
  return {
    id,
    name: "Assist sample",
    updatedAt: "2026-11-19T10:00:00.000Z",
    revision: 2,
    theme: { ...DEFAULT_TEMPLATE.palette },
    scenario: null,
    page: null,
    frames: [{ id: "desktop", name: "Desktop", device: "desktop", spec: page }],
    ...overrides,
  };
}

function firstFrame(document: DesignDocument) {
  const frame = document.frames[0];
  if (!frame) throw new Error("The test design needs a frame.");
  return frame;
}

function firstProposeCall(mock: { mock: { calls: Parameters<ProposalTransport>[] } }) {
  const call = mock.mock.calls[0];
  if (!call) throw new Error("Expected the proposal transport to be called.");
  return call;
}

function proposal(overrides: Partial<DesignProposal> = {}): DesignProposal {
  return {
    summary: "Headline refresh",
    operations: [
      { kind: "setText", nodeId: headline.id, text: "A sharper headline" },
      { kind: "setProp", nodeId: headline.id, key: "variant", value: "title" },
      {
        kind: "addLayer",
        parentId: "page",
        node: { type: "button", props: { text: "Continue" } },
      },
      { kind: "removeLayer", nodeId: headline.id },
      { kind: "moveLayer", nodeId: headline.id, parentId: "page" },
      { kind: "setTheme", key: "bg", value: "#ffffff" },
    ],
    revision: 3,
    operationId: "operation-1",
    state: "completed",
    provider: "local",
    model: "fixture-model",
    ...overrides,
  };
}

function renderAssist(
  options: {
    document?: DesignDocument;
    frame?: DesignDocument["frames"][number] | null;
    selected?: TemplateNode | null;
    scope?: AssistScope;
    onScope?: (scope: AssistScope) => void;
    onApply?: (
      value: StudioProposal,
    ) => { ok: true } | { ok: false; reason: "stale" | "invalid"; detail: string };
    onApplied?: (summary: string) => void;
    propose?: ProposalTransport;
  } = {},
) {
  const document = options.document ?? design();
  return renderWithProviders(
    <StudioAssist
      design={document}
      frame={options.frame === null ? undefined : (options.frame ?? firstFrame(document))}
      selected={options.selected === null ? undefined : (options.selected ?? headline)}
      scope={options.scope ?? "layer"}
      onScope={options.onScope ?? vi.fn()}
      onApply={options.onApply ?? vi.fn(() => ({ ok: true as const }))}
      onApplied={options.onApplied ?? vi.fn()}
      propose={options.propose ?? vi.fn<ProposalTransport>()}
    />,
    { withoutRouter: true },
  );
}

function promptInput(): HTMLTextAreaElement {
  return screen.getByLabelText(strings.workbench.studio.aiRequest);
}

function proposeButton(): HTMLElement {
  return screen.getByRole("button", { name: strings.workbench.studio.aiPropose });
}

describe("StudioAssist", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("submits a trimmed layer request and renders each patch operation for review", async () => {
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal());
    const onApply = vi.fn(() => ({ ok: true as const }));
    const onApplied = vi.fn();
    renderAssist({ propose, onApply, onApplied });

    expect(proposeButton()).toBeDisabled();
    fireEvent.change(promptInput(), { target: { value: "  Refine this title  " } });
    fireEvent.keyDown(promptInput(), { key: "Enter" });
    expect(propose).not.toHaveBeenCalled();
    fireEvent.keyDown(promptInput(), { key: "Enter", ctrlKey: true });

    await waitFor(() =>
      expect(screen.getByText((content) => content === "Headline refresh")).toBeInTheDocument(),
    );
    expect(propose).toHaveBeenCalledOnce();
    const [input, signal] = firstProposeCall(propose);
    expect(input).toMatchObject({
      prompt: "Refine this title",
      scope: { kind: "layer", frameId: "desktop", nodeId: headline.id },
      revision: 2,
    });
    expect(input.idempotencyKey).toBeTruthy();
    expect(signal).toBeInstanceOf(AbortSignal);
    expect(screen.getByText(strings.workbench.studio.aiOpText)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.aiOpProp)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.aiOpAdd)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.aiOpRemove)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.aiOpMove)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.aiOpTheme)).toBeInTheDocument();
    expect(
      screen.getByText((content) => content === "Current headline → A sharper headline"),
    ).toBeInTheDocument();
    expect(screen.getByText((content) => content === "local · fixture-model")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiApply }));
    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ ...proposal(), designId: "design-a" }),
    );
    expect(onApplied).toHaveBeenCalledWith("Headline refresh");
    expect(screen.queryByText((content) => content === "Headline refresh")).not.toBeInTheDocument();
    expect(promptInput()).toHaveValue("");
  });

  it("keeps incomplete and future operation previews readable", async () => {
    const longCopy = "A".repeat(70);
    const futureOperation = {
      kind: "futureOperation",
    } as unknown as DesignProposal["operations"][number];
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(
      proposal({
        operations: [
          { kind: "setText", nodeId: "missing-node", text: longCopy },
          { kind: "setProp", nodeId: "missing-node" },
          { kind: "addLayer", parentId: "missing-parent" },
          futureOperation,
        ],
      }),
    );
    renderAssist({ propose });

    fireEvent.change(promptInput(), { target: { value: "Preview a partial response" } });
    fireEvent.click(proposeButton());
    await screen.findByText((content) => content === "Headline refresh");

    expect(
      screen.getByText((content) => content === `— → ${longCopy.slice(0, 63)}…`),
    ).toBeInTheDocument();
    expect(screen.getByText((content) => content === "— → —")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "futureOperation")).toBeInTheDocument();
  });

  it.each([
    ["frame", { kind: "frame", frameId: "desktop" }],
    ["design", { kind: "design" }],
  ] as const)("sends the selected %s scope without layer identifiers", async (scope, payload) => {
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal({ operations: [] }));
    renderAssist({ scope, propose });

    fireEvent.change(promptInput(), { target: { value: "Improve the hierarchy" } });
    fireEvent.click(proposeButton());
    await waitFor(() => expect(propose).toHaveBeenCalledOnce());
    expect(firstProposeCall(propose)[0].scope).toEqual(payload);
  });

  it("requires a selected layer or active frame, and delegates scope changes", () => {
    const onScope = vi.fn();
    const { rerender } = renderAssist({ selected: null, scope: "layer", onScope });

    expect(screen.getByText(strings.workbench.studio.aiSelectLayer)).toBeInTheDocument();
    expect(proposeButton()).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiScopeFrame }));
    expect(onScope).toHaveBeenCalledWith("frame");

    const document = design();
    rerender(
      <StudioAssist
        design={document}
        frame={undefined}
        selected={undefined}
        scope="frame"
        onScope={onScope}
        onApply={() => ({ ok: true })}
        onApplied={vi.fn()}
        propose={vi.fn<ProposalTransport>()}
      />,
    );
    expect(proposeButton()).toBeDisabled();

    rerender(
      <StudioAssist
        design={document}
        frame={document.frames[0]}
        selected={undefined}
        scope="frame"
        onScope={onScope}
        onApply={() => ({ ok: true })}
        onApplied={vi.fn()}
        propose={vi.fn<ProposalTransport>()}
      />,
    );
    expect(proposeButton()).toBeDisabled();
    fireEvent.change(promptInput(), { target: { value: "A scoped request" } });
    expect(proposeButton()).toBeEnabled();
  });

  it.each([
    ["ai_unavailable", strings.workbench.studio.aiUnavailable],
    ["invalid_proposal", strings.workbench.studio.aiInvalid],
    ["conflict", strings.workbench.studio.aiConflict],
    ["invalid_argument", "Malformed request"],
    ["ai_failed", "The model failed"],
    ["unknown", "Unexpected response"],
  ] as const)("shows the safe recovery message for a %s proposal error", async (code, message) => {
    const propose = vi
      .fn<ProposalTransport>()
      .mockRejectedValue(new DesignProposalError(code, 503, message));
    renderAssist({ propose });

    fireEvent.change(promptInput(), { target: { value: "Try this" } });
    fireEvent.click(proposeButton());

    expect(await screen.findByRole("alert")).toHaveTextContent(message);
    expect(proposeButton()).toBeEnabled();
  });

  it("reports plain transport errors and ignores non-error rejection values safely", async () => {
    const propose = vi.fn<ProposalTransport>().mockRejectedValueOnce(new Error("network reset"));
    renderAssist({ propose });
    fireEvent.change(promptInput(), { target: { value: "Try this" } });
    fireEvent.click(proposeButton());
    expect(await screen.findByRole("alert")).toHaveTextContent("network reset");

    propose.mockRejectedValueOnce("offline");
    fireEvent.click(proposeButton());
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.workbench.studio.aiError);
  });

  it("falls back to a local idempotency key when secure random UUIDs are unavailable", async () => {
    const descriptor = Object.getOwnPropertyDescriptor(globalThis.crypto, "randomUUID");
    Object.defineProperty(globalThis.crypto, "randomUUID", {
      configurable: true,
      value: () => {
        throw new Error("secure context required");
      },
    });
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal({ operations: [] }));

    try {
      renderAssist({ propose });
      fireEvent.change(promptInput(), { target: { value: "Make a safe request" } });
      fireEvent.click(proposeButton());
      await waitFor(() => expect(propose).toHaveBeenCalledOnce());
      expect(firstProposeCall(propose)[0].idempotencyKey).toMatch(/^assist-/);
    } finally {
      if (descriptor) Object.defineProperty(globalThis.crypto, "randomUUID", descriptor);
      else Reflect.deleteProperty(globalThis.crypto, "randomUUID");
    }
  });

  it("marks proposals stale after composition changes but tolerates revision-only saves", async () => {
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal());
    const onApply = vi.fn(() => ({ ok: true as const }));
    const { rerender } = renderAssist({ propose, onApply });
    fireEvent.change(promptInput(), { target: { value: "Change the headline" } });
    fireEvent.click(proposeButton());
    await screen.findByText((content) => content === "Headline refresh");

    const saved = design("design-a", { revision: 3 });
    rerender(
      <StudioAssist
        design={saved}
        frame={firstFrame(saved)}
        selected={headline}
        scope="layer"
        onScope={vi.fn()}
        onApply={onApply}
        onApplied={vi.fn()}
        propose={propose}
      />,
    );
    expect(screen.queryByText(strings.workbench.studio.aiStale)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: strings.workbench.studio.aiApply })).toBeEnabled();

    const changedHeadline = { ...headline, props: { ...headline.props, text: "New composition" } };
    const changedDocument = design("design-a", {
      revision: 4,
      frames: [
        {
          ...firstFrame(saved),
          spec: { ...firstFrame(saved).spec, children: [changedHeadline] },
        },
      ],
    });
    rerender(
      <StudioAssist
        design={changedDocument}
        frame={firstFrame(changedDocument)}
        selected={changedHeadline}
        scope="layer"
        onScope={vi.fn()}
        onApply={onApply}
        onApplied={vi.fn()}
        propose={propose}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent(strings.workbench.studio.aiStale);
    expect(screen.getByRole("button", { name: strings.workbench.studio.aiApply })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiApply }));
    expect(onApply).not.toHaveBeenCalled();
  });

  it("clears a proposal on discard and reports an apply rejection without applying it", async () => {
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal());
    const onApply = vi.fn(() => ({
      ok: false as const,
      reason: "invalid" as const,
      detail: "Patch rejected",
    }));
    renderAssist({ propose, onApply });
    fireEvent.change(promptInput(), { target: { value: "Change the headline" } });
    fireEvent.click(proposeButton());
    await screen.findByText((content) => content === "Headline refresh");

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiApply }));
    expect(screen.getByRole("alert")).toHaveTextContent("Patch rejected");
    expect(screen.getByText((content) => content === "Headline refresh")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiDiscard }));
    expect(screen.queryByText((content) => content === "Headline refresh")).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("asks for a fresh proposal when application detects a stale race", async () => {
    const propose = vi.fn<ProposalTransport>().mockResolvedValue(proposal());
    const onApply = vi.fn(() => ({
      ok: false as const,
      reason: "stale" as const,
      detail: "unused",
    }));
    renderAssist({ propose, onApply });
    fireEvent.change(promptInput(), { target: { value: "Change the headline" } });
    fireEvent.click(proposeButton());
    await screen.findByText((content) => content === "Headline refresh");

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.studio.aiApply }));

    expect(screen.getByRole("alert")).toHaveTextContent(strings.workbench.studio.aiStale);
    expect(screen.getByText((content) => content === "Headline refresh")).toBeInTheDocument();
  });

  it("aborts pending work on a design switch and drops the late result", async () => {
    let resolveProposal!: (value: DesignProposal) => void;
    const propose = vi.fn<ProposalTransport>(
      () =>
        new Promise<DesignProposal>((resolve) => {
          resolveProposal = resolve;
        }),
    );
    const { rerender } = renderAssist({ propose });
    fireEvent.change(promptInput(), { target: { value: "Keep this on the original page" } });
    fireEvent.click(proposeButton());
    expect(await screen.findByRole("status")).toHaveTextContent(strings.workbench.studio.aiWaiting);
    const signal = firstProposeCall(propose)[1];

    const next = design("design-b");
    rerender(
      <StudioAssist
        design={next}
        frame={firstFrame(next)}
        selected={headline}
        scope="layer"
        onScope={vi.fn()}
        onApply={vi.fn(() => ({ ok: true as const }))}
        onApplied={vi.fn()}
        propose={propose}
      />,
    );
    await waitFor(() => expect(signal?.aborted).toBe(true));
    expect(screen.queryByRole("status")).not.toBeInTheDocument();

    resolveProposal(proposal());
    await waitFor(() =>
      expect(
        screen.queryByText((content) => content === "Headline refresh"),
      ).not.toBeInTheDocument(),
    );
    expect(propose).toHaveBeenCalledOnce();
  });
});
