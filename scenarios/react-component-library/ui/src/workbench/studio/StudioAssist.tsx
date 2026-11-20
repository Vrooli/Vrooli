import { useEffect, useRef, useState } from "react";
import { Sparkles, Wand2 } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../designs";
import { findNode, type TemplateNode } from "../templates/spec";
import { layerLabel } from "../templates/spec";
import { cx } from "../cx";
import { Segmented, WbButton } from "../primitives";
import { docSignature } from "./model";
import {
  DesignProposalError,
  type DesignPatchOperation,
  type ProposalTransport,
  type StudioProposal,
} from "./proposal";

export type AssistScope = "layer" | "frame" | "design";

/** A stable key without depending on `crypto.randomUUID` being present. */
function requestKey(): string {
  try {
    // Present in every browser this app supports; the fallback is for a
    // non-secure context, where `crypto.randomUUID` is unavailable.
    return globalThis.crypto.randomUUID();
  } catch {
    return `assist-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
  }
}

function findAnywhere(doc: DesignDocument, id: string | undefined): TemplateNode | undefined {
  if (!id) return undefined;
  for (const frame of doc.frames) {
    const found = findNode(frame.spec, id);
    if (found) return found;
  }
  return undefined;
}

function clip(value: string, limit = 64): string {
  return value.length > limit ? `${value.slice(0, limit - 1)}…` : value;
}

export function StudioAssist({
  design,
  frame,
  selected,
  scope,
  onScope,
  onApply,
  onApplied,
  propose,
}: {
  design: DesignDocument;
  frame: DesignFrame | undefined;
  selected: TemplateNode | undefined;
  scope: AssistScope;
  onScope: (scope: AssistScope) => void;
  onApply: (proposal: StudioProposal) => { ok: true } | { ok: false; reason: "stale" | "invalid"; detail: string };
  onApplied: (summary: string) => void;
  /** The transport for this surface: design studio or workspace studio. */
  propose: ProposalTransport;
}) {
  const { t } = useTranslation();
  // Literal t() calls, not a key table: the string catalog is built from the
  // callsites, and an indirect key would be pruned as an orphan.
  const scopeLabel: Record<AssistScope, string> = {
    layer: t("workbench.studio.aiScopeLayer", { defaultValue: "Selected layer" }),
    frame: t("workbench.studio.aiScopeFrame", { defaultValue: "Active display" }),
    design: t("workbench.studio.aiScopeDesign", { defaultValue: "Whole page" }),
  };
  const [prompt, setPrompt] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [proposal, setProposal] = useState<StudioProposal | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const designIdRef = useRef(design.id);
  designIdRef.current = design.id;

  useEffect(() => () => abortRef.current?.abort(), []);

  // Switching designs invalidates both the request and any proposal already
  // on screen: its layer ids belong to the document that was open.
  useEffect(() => {
    abortRef.current?.abort();
    setProposal(null);
    setError(null);
    setBusy(false);
  }, [design.id]);

  const scopeReady =
    scope === "design" ||
    (scope === "frame" && Boolean(frame)) ||
    (scope === "layer" && Boolean(selected));

  const submit = () => {
    const trimmed = prompt.trim();
    if (!trimmed || busy || !scopeReady) return;
    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;
    setBusy(true);
    setError(null);
    setProposal(null);
    const designId = design.id;
    const signature = docSignature(design);
    const scopePayload =
      scope === "layer"
        ? { kind: "layer" as const, frameId: frame?.id, nodeId: selected?.id }
        : scope === "frame"
          ? { kind: "frame" as const, frameId: frame?.id }
          : { kind: "design" as const };
    void propose(
      {
        prompt: trimmed,
        scope: scopePayload,
        revision: design.revision,
        idempotencyKey: requestKey(),
      },
      controller.signal,
    )
      .then((result) => {
        // A result that arrives after a tab switch belongs to a document that
        // is no longer on screen; drop it instead of offering the wrong patch.
        if (controller.signal.aborted || designId !== designIdRef.current) return;
        setProposal({ ...result, signature, designId });
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        setError(copyForError(cause));
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
  };

  // The document signature, not the revision: autosave can bump the revision
  // while a proposal is in flight without changing the composition.
  const stale = Boolean(
    proposal && (proposal.designId !== design.id || proposal.signature !== docSignature(design)),
  );

  const apply = () => {
    if (!proposal) return;
    const result = onApply(proposal);
    if (result.ok) {
      onApplied(proposal.summary);
      setProposal(null);
      setPrompt("");
      setError(null);
      return;
    }
    setError(
      result.reason === "stale"
        ? t("workbench.studio.aiStale", {
            defaultValue: "The design changed after this proposal. Ask again for a fresh one.",
          })
        : result.detail,
    );
  };

  const subject =
    scope === "layer"
      ? selected
        ? layerLabel(selected)
        : t("workbench.studio.nothing", { defaultValue: "Nothing selected" })
      : scope === "frame"
        ? (frame?.name ?? t("workbench.studio.nothing", { defaultValue: "Nothing selected" }))
        : design.name;

  const describe = (operation: DesignPatchOperation): { title: string; detail?: string } => {
    const target = findAnywhere(design, operation.nodeId);
    const parent = findAnywhere(design, operation.parentId);
    // A person reads layer names, not node ids. The id stays the identity the
    // patch uses; only the description is humanised.
    const nameOf = (node: TemplateNode | undefined, id: string | undefined) =>
      node ? layerLabel(node) : (id ?? "");
    switch (operation.kind) {
      case "setText": {
        const before = typeof target?.props.text === "string" ? target.props.text : "";
        return {
          title: t("workbench.studio.aiOpText", {
            defaultValue: "Rewrite copy in {{layer}}",
            layer: nameOf(target, operation.nodeId),
          }),
          detail: `${clip(before) || "—"} → ${clip(operation.text ?? "")}`,
        };
      }
      case "setProp": {
        const before = target?.props[operation.key ?? ""];
        return {
          title: t("workbench.studio.aiOpProp", {
            defaultValue: "Set {{key}} on {{layer}}",
            key: operation.key ?? "",
            layer: nameOf(target, operation.nodeId),
          }),
          detail: `${String(before ?? "—")} → ${String(operation.value ?? "—")}`,
        };
      }
      case "addLayer":
        return {
          title: t("workbench.studio.aiOpAdd", {
            defaultValue: "Add {{type}} inside {{layer}}",
            type: operation.node?.type ?? "",
            layer: nameOf(parent, operation.parentId),
          }),
        };
      case "removeLayer":
        return {
          title: t("workbench.studio.aiOpRemove", {
            defaultValue: "Remove {{layer}}",
            layer: nameOf(target, operation.nodeId),
          }),
          detail: target?.type,
        };
      case "moveLayer":
        return {
          title: t("workbench.studio.aiOpMove", {
            defaultValue: "Move {{layer}} into {{parent}}",
            layer: nameOf(target, operation.nodeId),
            parent: nameOf(parent, operation.parentId),
          }),
        };
      case "setTheme":
        return {
          title: t("workbench.studio.aiOpTheme", {
            defaultValue: "Set {{key}}",
            key: operation.key ?? "",
          }),
          detail: String(operation.value ?? ""),
        };
      default:
        return { title: String(operation.kind) };
    }
  };

  const copyForError = (cause: unknown): string => {
    if (!(cause instanceof DesignProposalError)) {
      return cause instanceof Error
        ? cause.message
        : t("workbench.studio.aiError", { defaultValue: "The proposal could not be requested." });
    }
    switch (cause.code) {
      case "ai_unavailable":
        return t("workbench.studio.aiUnavailable", {
          defaultValue:
            "The AI gateway is not reachable. Start it with `vrooli scenario start ai-gateway`, then try again.",
        });
      case "ai_failed":
        return cause.message;
      case "invalid_proposal":
        return t("workbench.studio.aiInvalid", {
          defaultValue:
            "The model proposed a patch this design cannot apply. Try a narrower request or another scope.",
        });
      case "conflict":
        return t("workbench.studio.aiConflict", {
          defaultValue: "This design was saved elsewhere. Reload the studio before asking again.",
        });
      default:
        return cause.message;
    }
  };

  return (
    <div className="studio-assist">
      <div className="studio-ai-banner">
        <Sparkles aria-hidden size={14} />
        <span>
          <b>{t("workbench.studio.aiTitle", { defaultValue: "Propose a change" })}</b>
          <small>
            {t("workbench.studio.aiBody", {
              defaultValue:
                "Runs locally through ai-gateway. The answer is a patch you can inspect, apply as one undoable step, or discard.",
            })}
          </small>
        </span>
      </div>

      <div className="studio-inspector-section">
        <span className="wb-caption">{t("workbench.studio.aiScope", { defaultValue: "Context to send" })}</span>
        <Segmented
          label={t("workbench.studio.aiScope", { defaultValue: "Context to send" })}
          value={scope}
          onChange={onScope}
          items={(Object.keys(scopeLabel) as AssistScope[]).map((id) => ({
            id,
            label: scopeLabel[id],
          }))}
        />
        <div className="studio-kv">
          <span>{t("workbench.studio.aiSubject", { defaultValue: "subject" })}</span>
          <b>{subject}</b>
        </div>
      </div>

      <div className="studio-inspector-section">
        <label className="studio-ai-label" htmlFor="studio-ai-prompt">
          {t("workbench.studio.aiRequest", { defaultValue: "What should change?" })}
        </label>
        <textarea
          id="studio-ai-prompt"
          className="studio-ai-input"
          rows={4}
          value={prompt}
          maxLength={2000}
          placeholder={t("workbench.studio.aiPlaceholder", {
            defaultValue: "Make the headline specific to this page and soften the secondary action.",
          })}
          onChange={(event) => setPrompt(event.target.value)}
          onKeyDown={(event) => {
            if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
              event.preventDefault();
              submit();
            }
          }}
        />
        <div className="studio-ai-actions">
          <WbButton
            variant="primary"
            size="md"
            disabled={busy || !prompt.trim() || !scopeReady}
            onClick={submit}
          >
            <Wand2 aria-hidden size={13} />
            {busy
              ? t("workbench.studio.aiProposing", { defaultValue: "Proposing…" })
              : t("workbench.studio.aiPropose", { defaultValue: "Propose a patch" })}
          </WbButton>
          <span className="studio-ai-hint">
            {t("workbench.studio.aiShortcut", { defaultValue: "⌘/Ctrl + Enter" })}
          </span>
        </div>
        {!scopeReady ? (
          <p className="studio-note">
            {t("workbench.studio.aiSelectLayer", {
              defaultValue: "Select a layer to scope the request to it.",
            })}
          </p>
        ) : null}
        {busy ? (
          <p className="studio-note" role="status">
            {t("workbench.studio.aiWaiting", {
              defaultValue: "A local model can take up to a minute. Nothing is applied until you accept the patch.",
            })}
          </p>
        ) : null}
        {error ? (
          <p className="studio-ai-error" role="alert">
            {error}
          </p>
        ) : null}
      </div>

      {proposal ? (
        <section className="studio-ai-proposal" aria-label={t("workbench.studio.aiProposal", { defaultValue: "Proposed patch" })}>
          <div className="studio-ai-proposal-head">
            <b>{proposal.summary}</b>
            <span className="studio-ai-model">
              {[proposal.provider, proposal.model].filter(Boolean).join(" · ")}
            </span>
          </div>
          <ol className="studio-ai-ops">
            {proposal.operations.map((operation, index) => {
              const copy = describe(operation);
              return (
                <li key={`${operation.kind}-${index}`} className={cx("studio-ai-op", `is-${operation.kind}`)}>
                  <span>{copy.title}</span>
                  {copy.detail ? <small>{copy.detail}</small> : null}
                </li>
              );
            })}
          </ol>
          {stale ? (
            <p className="studio-ai-error" role="status">
              {t("workbench.studio.aiStale", {
                defaultValue: "The design changed after this proposal. Ask again for a fresh one.",
              })}
            </p>
          ) : null}
          <div className="studio-ai-actions">
            <WbButton variant="primary" size="md" disabled={stale} onClick={apply}>
              {t("workbench.studio.aiApply", { defaultValue: "Apply as one step" })}
            </WbButton>
            <button
              type="button"
              className="studio-ai-discard"
              onClick={() => {
                setProposal(null);
                setError(null);
              }}
            >
              {t("workbench.studio.aiDiscard", { defaultValue: "Discard" })}
            </button>
          </div>
          <p className="studio-note">
            {t("workbench.studio.aiUndo", {
              defaultValue: "Applied patches are one entry in the history stack, so Undo reverses the whole change.",
            })}
          </p>
        </section>
      ) : null}
    </div>
  );
}

