import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";

import { conversationsClient } from "../../../api/notifications";
import type { PacketDraft } from "../model/draft";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

function sha256(bytes: Uint8Array): Promise<string> {
  const buffer = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(buffer).set(bytes);
  return crypto.subtle.digest("SHA-256", buffer).then((digest) => [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join(""));
}

export function AnswerAttachments({ images, attachments, disabled, onChange, onBusyChange }: {
  images: PacketDraft["answerImages"];
  attachments: PacketDraft["answerAttachments"];
  disabled: boolean;
  onChange: (update: (current: PacketDraft) => PacketDraft) => void;
  onBusyChange?: (busy: boolean) => void;
}) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const setUploading = (next: boolean) => { setBusy(next); onBusyChange?.(next); };
  const addFiles = async (files: FileList | null) => {
    if (!files?.length) return;
    setError("");
    setUploading(true);
    try {
      const selected = Array.from(files);
      if (images.length + attachments.length + selected.length > 8) throw new Error(t(strings.packet.responseFiles.tooMany));
      const total = [...images.map((image) => image.sizeBytes), ...attachments.map((file) => file.sizeBytes), ...selected.map((file) => file.size)].reduce((sum, size) => sum + size, 0);
      if (total > 24 * 1024 * 1024) throw new Error(t(strings.packet.responseFiles.tooLarge));
      for (const file of selected) if (file.size <= 0 || file.size > 12 * 1024 * 1024) throw new Error(t(strings.packet.responseFiles.singleTooLarge, { name: file.name }));
      const uploaded: PacketDraft["answerImages"] = [];
      const uploadedFiles: PacketDraft["answerAttachments"] = [];
      for (const file of selected) {
        const bytes = new Uint8Array(await file.arrayBuffer());
        const hash = await sha256(bytes);
        const mediaType = file.type || "application/octet-stream";
        await conversationsClient.putAttachmentBlob({ sha256: hash, mediaType, content: bytes });
        const safeName = file.name.normalize("NFKD").replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 96) || `file-${uploaded.length + uploadedFiles.length + 1}`;
        const metadata = { name: safeName, sha256: hash, mediaType, sizeBytes: file.size };
        if (file.type.startsWith("image/")) uploaded.push(metadata);
        else uploadedFiles.push(metadata);
      }
      onChange((current) => ({ ...current, answerImages: [...current.answerImages, ...uploaded], answerAttachments: [...current.answerAttachments, ...uploadedFiles] }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.packet.responseFiles.failed));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1 text-sm font-medium">
        {t(strings.packet.responseFiles.attach)}
        <input type="file" multiple disabled={disabled || busy} onChange={(event) => { void addFiles(event.currentTarget.files); event.currentTarget.value = ""; }} className="rounded-control border border-app-border bg-app-surface p-2 text-sm" />
      </label>
      {(images.length > 0 || attachments.length > 0) && <ul aria-label={t(strings.packet.responseFiles.listLabel)} className="flex flex-col gap-2">
        {[...images.map((file) => ({ ...file, image: true })), ...attachments.map((file) => ({ ...file, image: false }))].map((file) => <li key={`${file.name}:${file.sha256}`} className="flex items-center justify-between gap-2 rounded-control border border-app-border px-3 py-2 text-sm">
          <span className="min-w-0 truncate">{file.name}</span>
          <Button type="button" size="sm" variant="ghost" disabled={disabled || busy} aria-label={t(strings.packet.responseFiles.remove, { name: file.name })} onClick={() => onChange((current) => file.image
            ? { ...current, answerImages: current.answerImages.filter((item) => item.sha256 !== file.sha256) }
            : { ...current, answerAttachments: current.answerAttachments.filter((item) => item.sha256 !== file.sha256) })}>{t(strings.packet.responseFiles.remove, { name: file.name })}</Button>
        </li>)}
      </ul>}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
    </div>
  );
}
