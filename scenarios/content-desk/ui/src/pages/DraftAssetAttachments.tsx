import { useCallback, useEffect, useState } from "react";

import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";

import { artifactsClient } from "../api/artifacts";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type Attachment = { id: string; assetId: string; role: string; aspectRatio: string; altText: string; position: number };

export function DraftAssetAttachments({ draft }: { draft: Draft }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [assetId, setAssetId] = useState("");
  const [role, setRole] = useState("hero");
  const [aspectRatio, setAspectRatio] = useState("16:9");
  const [altText, setAltText] = useState("");
  const [position, setPosition] = useState("0");
  const [state, setState] = useState<"idle" | "loading" | "saving" | "error">("idle");
  const [message, setMessage] = useState("");

  const loadAttachments = useCallback(async (): Promise<boolean> => {
    setState("loading");
    try {
      const response = await artifactsClient.listDraftAttachments({ draftId: draft.id });
      setAttachments(response.attachments.map((item) => ({ id: item.id, assetId: item.assetId, role: item.role, aspectRatio: item.aspectRatio, altText: item.altText, position: item.position })));
      setState("idle");
      return true;
    } catch {
      setState("error");
      setMessage(t(strings.pages.content.assetLoadError));
      return false;
    }
  }, [draft.id, t]);

  useEffect(() => {
    if (open) void loadAttachments();
  }, [loadAttachments, open]);

  async function attach() {
    setState("saving");
    setMessage("");
    try {
      await artifactsClient.attachReleasedAsset({ draftId: draft.id, assetId: assetId.trim(), role: role.trim(), aspectRatio: aspectRatio.trim(), altText: altText.trim(), position: positionValue });
      setAssetId("");
      setAltText("");
      if (!await loadAttachments()) return;
      setMessage(t(strings.pages.content.assetAttached));
    } catch (caught) {
      setState("error");
      setMessage(caught instanceof Error ? caught.message : t(strings.pages.content.assetAttachError));
    }
  }

  const positionValue = Number(position);
  const validPosition = Number.isSafeInteger(positionValue) && positionValue >= 0;
  const canAttach = draft.status !== "published" && draft.status !== "abandoned" && validPosition && Boolean(assetId.trim() && role.trim() && aspectRatio.trim() && altText.trim());

  return <div className="mt-3">
    <Button size="sm" type="button" variant="secondary" aria-expanded={open} onClick={() => setOpen((value) => !value)}>{t(open ? strings.pages.content.hideAssetReferences : strings.pages.content.manageAssetReferences)}</Button>
    {open ? <section aria-label={t(strings.pages.content.assetReferences)} className="mt-3 rounded-control border border-app-border p-3 text-sm">
      <h4 className="font-medium">{t(strings.pages.content.assetReferences)}</h4>
      <p className="mt-1 text-app-muted-foreground">{t(strings.pages.content.assetReferencesIntro)}</p>
      {state === "loading" ? <p aria-live="polite" className="mt-2 text-app-muted-foreground">{t(strings.pages.content.loadingAssets)}</p> : null}
      {attachments.length > 0 ? <ul className="mt-2 space-y-1">{attachments.map((item) => <li key={item.id}>{item.position} · {item.assetId} · {item.role} · {item.aspectRatio} · {item.altText}</li>)}</ul> : state === "idle" ? <p className="mt-2 text-app-muted-foreground">{t(strings.pages.content.noAssetReferences)}</p> : null}
      {state === "error" ? <p role="alert" className="mt-2 text-app-danger">{message}</p> : null}
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        <label className="grid gap-1"><span>{t(strings.pages.content.assetId)}</span><Input value={assetId} onChange={(event) => setAssetId(event.target.value)} /></label>
        <label className="grid gap-1"><span>{t(strings.pages.content.assetRole)}</span><Input value={role} onChange={(event) => setRole(event.target.value)} /></label>
        <label className="grid gap-1"><span>{t(strings.pages.content.assetAspectRatio)}</span><Input value={aspectRatio} onChange={(event) => setAspectRatio(event.target.value)} /></label>
        <label className="grid gap-1"><span>{t(strings.pages.content.assetPosition)}</span><Input type="number" min="0" step="1" value={position} onChange={(event) => setPosition(event.target.value)} /></label>
      </div>
      <label className="mt-2 grid gap-1"><span>{t(strings.pages.content.assetAltText)}</span><Input value={altText} onChange={(event) => setAltText(event.target.value)} /></label>
      <Button className="mt-3" size="sm" type="button" variant="secondary" disabled={!canAttach || state === "saving" || state === "loading"} onClick={() => void attach()}>{t(strings.pages.content.attachReleasedAsset)}</Button>
      {state === "idle" && message ? <p role="status" className="mt-2 text-app-primary">{message}</p> : null}
    </section> : null}
  </div>;
}
