import { useQuery } from "@tanstack/react-query";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { fetchScenarioExternalUrl } from "../../api/client";
import { hasLocalFileReference } from "./decisionLinks";

/** The text fields of a decision that may name local files. */
export function decisionTexts(ask: Ask): string[] {
  return [ask.question, ask.recommendationReason, ask.bigPicture, ask.technicalDetails, ask.note, ask.contextUrl, ...ask.responses.map((response) => response.text), ...ask.attachmentNotes.map((note) => note.note)];
}

/**
 * Web Console's address, only when the decision names a local file the Hub
 * cannot open itself: files attached to the decision open in the Hub's own
 * viewer, so the link target is looked up only for the rest.
 */
export function useWebConsole(ask: Ask) {
  const needed = decisionTexts(ask).some((text) => hasLocalFileReference(text, ask.attachments));
  const query = useQuery({
    queryKey: ["scenario-external-url", "web-console"],
    enabled: needed,
    staleTime: 5 * 60_000,
    retry: 1,
    retryDelay: 200,
    queryFn: async () => {
      const url = new URL(await fetchScenarioExternalUrl("web-console"));
      if ((url.protocol !== "https:" && url.protocol !== "http:") || url.username || url.password) {
        throw new Error("Web Console returned an unusable URL");
      }
      return url.href;
    },
  });
  return { needed, query };
}
