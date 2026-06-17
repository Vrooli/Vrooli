import { useEffect, useRef, useState, type ChangeEvent, type DragEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send, Upload, X } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatBytes } from "../../lib/formatBytes";
import { errorMessage } from "../../lib/errorMessage";
import { useSession } from "../session/SessionProvider";
import { useDevicesQuery } from "../devices/queries";
import { transferClient, uploadItem, Retention } from "../../api/transfer";
import { ITEMS_QUERY_KEY } from "./queries";
import {
  stageFile,
  stageText,
  type StagedItem,
} from "./staging";

const RETENTION_CHOICES: readonly Retention[] = [Retention.LIVE, Retention.HELD, Retention.PINNED];
const RETENTION_LABEL = {
  [Retention.UNSPECIFIED]: strings.transfer.retention.held,
  [Retention.LIVE]: strings.transfer.retention.live,
  [Retention.HELD]: strings.transfer.retention.held,
  [Retention.PINNED]: strings.transfer.retention.pinned,
} as const satisfies Record<Retention, string>;

/**
 * The Send half (bottom). Stage files (drop or pick) and text snippets as
 * cards, set per-item retention + an optional target device (when the owner is
 * signed in, otherwise broadcast-only), then push: text via the Connect
 * `createTextItem`, files via the device-token multipart upload. On success the
 * items query is invalidated so the receive panel reflects the new item.
 */
export function SendPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { isOwner } = useSession();
  const devicesQuery = useDevicesQuery(isOwner);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [staged, setStaged] = useState<StagedItem[]>([]);
  const [draftText, setDraftText] = useState("");
  const [dragging, setDragging] = useState(false);
  const [status, setStatus] = useState<"idle" | "sent">("idle");

  // Revoke any image preview object URLs when the staged set changes / unmounts.
  useEffect(() => {
    return () => {
      staged.forEach((item) => {
        if (item.kind === "file" && item.previewUrl) URL.revokeObjectURL(item.previewUrl);
      });
    };
  }, [staged]);

  const addFiles = (files: FileList | File[]) => {
    const next = Array.from(files).map(stageFile);
    if (next.length > 0) setStaged((prev) => [...prev, ...next]);
  };

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.files) addFiles(event.target.files);
    event.target.value = "";
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragging(false);
    if (event.dataTransfer.files.length > 0) addFiles(event.dataTransfer.files);
  };

  const handleAddText = () => {
    const text = draftText.trim();
    if (!text) return;
    setStaged((prev) => [...prev, stageText(text)]);
    setDraftText("");
  };

  const updateStaged = (id: string, patch: Partial<Pick<StagedItem, "retention" | "targetDeviceId">>) => {
    setStaged((prev) => prev.map((item) => (item.id === id ? { ...item, ...patch } : item)));
  };

  const removeStaged = (id: string) => {
    setStaged((prev) => {
      const target = prev.find((item) => item.id === id);
      if (target?.kind === "file" && target.previewUrl) URL.revokeObjectURL(target.previewUrl);
      return prev.filter((item) => item.id !== id);
    });
  };

  const sendMutation = useMutation({
    mutationFn: async (items: StagedItem[]) => {
      for (const item of items) {
        if (item.kind === "text") {
          await transferClient.createTextItem({
            text: item.text,
            name: "",
            retention: item.retention,
            targetDeviceId: item.targetDeviceId,
          });
        } else {
          await uploadItem(item.file, {
            retention: item.retention,
            targetDeviceId: item.targetDeviceId,
          });
        }
      }
    },
    onSuccess: () => {
      setStaged([]);
      setStatus("sent");
      void queryClient.invalidateQueries({ queryKey: ITEMS_QUERY_KEY });
      window.setTimeout(() => setStatus("idle"), 2000);
    },
  });

  const trustedDevices = (devicesQuery.data ?? []).filter((d) => d.id);

  return (
    <section
      data-testid={selectors.send.panel}
      aria-labelledby="send-heading"
      className="flex h-full min-h-0 flex-col gap-3 border-t-4 border-app-accent bg-app-surface p-4"
    >
      <header>
        <h2 id="send-heading" className="text-lg font-semibold text-app-accent">
          {t(strings.transfer.send.heading)}
        </h2>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.transfer.send.description)}
        </p>
      </header>

      <div className="grid min-h-0 flex-1 gap-3 overflow-auto md:grid-cols-2">
        <div className="flex flex-col gap-3">
          {/* The drop target is a non-interactive region: drag/drop is a
              progressive enhancement, and the keyboard/click-equivalent path is
              the real Browse button + file input below (so we avoid an
              interactive role that would wrap focusable children and trip axe).
              The drag listeners on a non-interactive element are the accepted
              drop-zone pattern; the rule is disabled for this line only. */}
          {/* eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions */}
          <div
            data-testid={selectors.send.dropZone}
            role="group"
            aria-label={t(strings.transfer.send.fileInputLabel)}
            onDragOver={(e) => {
              e.preventDefault();
              setDragging(true);
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={handleDrop}
            className={[
              "flex flex-col items-center justify-center gap-2 rounded-panel border-2 border-dashed p-6 text-center text-sm transition-colors",
              dragging ? "border-app-accent bg-app-surface-muted" : "border-app-border",
            ].join(" ")}
          >
            <Upload aria-hidden="true" className="h-6 w-6 text-app-muted-foreground" />
            <p className="text-app-muted-foreground">{t(strings.transfer.send.dropHint)}</p>
            <Button variant="outline" size="sm" onClick={() => fileInputRef.current?.click()}>
              {t(strings.transfer.send.browse)}
            </Button>
            <input
              ref={fileInputRef}
              data-testid={selectors.send.fileInput}
              type="file"
              multiple
              className="sr-only"
              aria-label={t(strings.transfer.send.fileInputLabel)}
              onChange={handleFileChange}
            />
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="send-text" className="text-xs font-medium text-app-muted-foreground">
              {t(strings.transfer.send.textLabel)}
            </label>
            <Textarea
              id="send-text"
              data-testid={selectors.send.textInput}
              value={draftText}
              onChange={(e) => setDraftText(e.target.value)}
              placeholder={t(strings.transfer.send.textPlaceholder)}
            />
            <Button
              data-testid={selectors.send.addText}
              variant="outline"
              size="sm"
              className="w-fit"
              onClick={handleAddText}
              disabled={draftText.trim().length === 0}
              title={draftText.trim().length === 0 ? t(strings.transfer.send.needText) : undefined}
            >
              {t(strings.transfer.send.addText)}
            </Button>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <h3 className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.transfer.send.stagedHeading)}
          </h3>
          {staged.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.transfer.send.stagedEmpty)}
            </p>
          ) : (
            <ul data-testid={selectors.send.staged} className="flex flex-col gap-2">
              {staged.map((item) => (
                <li
                  key={item.id}
                  data-testid={selectors.send.stagedItem({ id: item.id })}
                  className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3"
                >
                  <div className="flex items-start gap-3">
                    {item.kind === "file" && item.previewUrl ? (
                      <img
                        src={item.previewUrl}
                        alt={item.file.name}
                        className="h-10 w-10 shrink-0 rounded-control object-cover"
                      />
                    ) : null}
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-app-foreground">
                        {item.kind === "file" ? item.file.name : item.text.slice(0, 60)}
                      </p>
                      {item.kind === "file" && (
                        <p className="text-xs text-app-muted-foreground">{formatBytes(item.file.size)}</p>
                      )}
                    </div>
                    <Button
                      data-testid={selectors.send.removeStaged({ id: item.id })}
                      variant="outline"
                      size="sm"
                      onClick={() => removeStaged(item.id)}
                      aria-label={t(strings.transfer.send.removeStaged)}
                    >
                      <X aria-hidden="true" className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <label className="flex items-center gap-1 text-xs text-app-muted-foreground">
                      {t(strings.transfer.send.retentionLabel)}
                      <select
                        value={item.retention}
                        onChange={(e) => updateStaged(item.id, { retention: Number(e.target.value) })}
                        className="rounded-control border border-app-border bg-app-surface px-1.5 py-1 text-app-foreground"
                      >
                        {RETENTION_CHOICES.map((r) => (
                          <option key={r} value={r}>
                            {t(RETENTION_LABEL[r])}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="flex items-center gap-1 text-xs text-app-muted-foreground">
                      {t(strings.transfer.send.targetLabel)}
                      <select
                        value={item.targetDeviceId}
                        onChange={(e) => updateStaged(item.id, { targetDeviceId: e.target.value })}
                        disabled={trustedDevices.length === 0}
                        className="rounded-control border border-app-border bg-app-surface px-1.5 py-1 text-app-foreground"
                      >
                        <option value="">{t(strings.transfer.send.targetBroadcast)}</option>
                        {trustedDevices.map((d) => (
                          <option key={d.id} value={d.id}>
                            {d.name}
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <footer className="flex flex-wrap items-center gap-3">
        <Button
          data-testid={selectors.send.sendButton}
          onClick={() => sendMutation.mutate(staged)}
          disabled={staged.length === 0 || sendMutation.isPending}
        >
          <Send aria-hidden="true" className="me-2 h-4 w-4" />
          {sendMutation.isPending
            ? t(strings.transfer.send.sending)
            : t(strings.transfer.send.send, { count: staged.length })}
        </Button>
        {(status === "sent" || sendMutation.error) && (
          <p data-testid={selectors.send.status} className={sendMutation.error ? "text-sm text-app-danger" : "text-sm text-app-success"}>
            {sendMutation.error ? errorMessage(sendMutation.error, t) : t(strings.transfer.send.sent)}
          </p>
        )}
      </footer>
    </section>
  );
}
