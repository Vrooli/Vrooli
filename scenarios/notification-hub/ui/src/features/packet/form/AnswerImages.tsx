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

export function AnswerImages({ images, disabled, onChange, onBusyChange }: {
  images: PacketDraft["answerImages"];
  disabled: boolean;
  onChange: (update: (current: PacketDraft) => PacketDraft) => void;
  onBusyChange?: (busy: boolean) => void;
}) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const setUploading = (next: boolean) => { setBusy(next); onBusyChange?.(next); };
  const addImages = async (files: FileList | null) => {
    if (!files?.length) return;
    setError("");
    setUploading(true);
    try {
      const selected = Array.from(files);
      if (images.length + selected.length > 8) throw new Error(t(strings.packet.images.tooMany));
      const total = [...images.map((image) => image.sizeBytes), ...selected.map((file) => file.size)].reduce((sum, size) => sum + size, 0);
      if (total > 24 * 1024 * 1024) throw new Error(t(strings.packet.images.tooLarge));
      for (const file of selected) if (!file.type.startsWith("image/") || file.size <= 0 || file.size > 12 * 1024 * 1024) throw new Error(t(strings.packet.images.singleTooLarge, { name: file.name }));
      const uploaded: PacketDraft["answerImages"] = [];
      for (const file of selected) {
        const bytes = new Uint8Array(await file.arrayBuffer());
        const hash = await sha256(bytes);
        await conversationsClient.putAttachmentBlob({ sha256: hash, mediaType: file.type, content: bytes });
        uploaded.push({ name: file.name.slice(0, 96) || `image-${uploaded.length + 1}`, sha256: hash, mediaType: file.type, sizeBytes: file.size });
      }
      onChange((current) => ({ ...current, answerImages: [...current.answerImages, ...uploaded] }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.packet.images.failed));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1 text-sm font-medium">
        {t(strings.packet.images.attach)}
        <input type="file" accept="image/*" multiple disabled={disabled || busy} onChange={(event) => { void addImages(event.currentTarget.files); event.currentTarget.value = ""; }} className="rounded-control border border-app-border bg-app-surface p-2 text-sm" />
      </label>
      {images.length > 0 && <ul aria-label={t(strings.packet.images.listLabel)} className="flex flex-col gap-2">
        {images.map((image) => <li key={`${image.name}:${image.sha256}`} className="flex items-center justify-between gap-2 rounded-control border border-app-border px-3 py-2 text-sm">
          <span className="truncate">{image.name}</span>
          <Button type="button" size="sm" variant="ghost" disabled={disabled || busy} aria-label={t(strings.packet.images.remove, { name: image.name })} onClick={() => onChange((current) => ({ ...current, answerImages: current.answerImages.filter((item) => item.sha256 !== image.sha256) }))}>{t(strings.packet.images.remove, { name: image.name })}</Button>
        </li>)}
      </ul>}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
    </div>
  );
}
