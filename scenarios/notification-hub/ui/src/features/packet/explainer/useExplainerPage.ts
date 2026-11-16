import { useQuery } from "@tanstack/react-query";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { AskAttachment_Kind } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { fetchAttachment } from "../../../api/attachments";
import {
  findReferences,
  planInlining,
  type ExplainerAttachment,
  type ReferenceTarget,
} from "./document";

export interface ExplainerPage {
  html: string;
  dataUrls: ReadonlyMap<string, string>;
  deferred: readonly ReferenceTarget[];
}

export function toExplainerAttachment(attachment: AskAttachment): ExplainerAttachment {
  return {
    name: attachment.name,
    folder: attachment.kind === AskAttachment_Kind.FOLDER,
    sizeBytes: Number(attachment.sizeBytes),
    mediaType: attachment.mediaType,
    entries: attachment.entries.map((entry) => ({ path: entry.path, sizeBytes: Number(entry.sizeBytes), mediaType: entry.mediaType })),
  };
}

function blobToDataUrl(blob: Blob, mediaType: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error ?? new Error("could not read attachment"));
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.readAsDataURL(blob.type ? blob : new Blob([blob], { type: mediaType }));
  });
}

/**
 * Fetch the explainer page and the attachments it names. The page is read from
 * the Hub's own copy (never a host path). Files that fit the inline budget
 * become `data:` URLs; the rest, and any that fail to load, stay placeholders
 * that open the viewer. Only the page itself failing is an error, and that
 * never blocks the decision.
 */
export function useExplainerPage(askId: string, explainer: AskAttachment | undefined, attachments: readonly AskAttachment[]) {
  const others = attachments.filter((attachment) => attachment.name !== explainer?.name);
  const fingerprint = explainer ? [explainer, ...others].map((attachment) => `${attachment.name}:${attachment.sha256}:${attachment.entries.length}`).join("|") : "kit-template";
  return useQuery({
    queryKey: ["ask-explainer", askId, fingerprint],
    enabled: Boolean(explainer),
    staleTime: Infinity,
    retry: 1,
    queryFn: async ({ signal }): Promise<ExplainerPage> => {
      if (!explainer) throw new Error("No attached explainer page is available.");
      const html = await (await fetchAttachment(askId, explainer.name, "", signal)).text();
      const { inline, deferred } = planInlining(findReferences(html, others.map(toExplainerAttachment)));
      const dataUrls = new Map<string, string>();
      const failed: ReferenceTarget[] = [];
      await Promise.all(
        inline.map(async (reference) => {
          try {
            const blob = await (await fetchAttachment(askId, reference.name, reference.path, signal)).blob();
            dataUrls.set(reference.value, await blobToDataUrl(blob, reference.mediaType));
          } catch {
            failed.push(reference);
          }
        }),
      );
      return { html, dataUrls, deferred: [...deferred, ...failed] };
    },
  });
}
