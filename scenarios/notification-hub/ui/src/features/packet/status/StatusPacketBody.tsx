import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { useClaimNavSection } from "../../../layout/navSection";
import { usePacketDraft } from "../usePacketDraft";
import { PacketBack } from "../PacketBack";
import { PacketCover } from "../PacketCover";
import { withPrefill } from "../model/answer";
import { usePacketOrigin } from "../usePacketOrigin";
import { ExplainerFrame } from "../explainer/ExplainerFrame";
import { StatusAnswerBar } from "./StatusAnswerBar";
import { explainerOf } from "../viewer/attachments";

export function StatusPacketBody({ ask, subject, signedIn }: { ask: Ask; subject: string; signedIn: boolean }) {
  const { returnTo, section } = usePacketOrigin(ask);
  useClaimNavSection(section);
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, updateDraft] = usePacketDraft(subject, ask.id);
  const explainer = explainerOf(ask.attachments);
  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["ask", ask.id, subject] });
    void queryClient.invalidateQueries({ queryKey: ["asks"] });
  }, [ask.id, queryClient, subject]);
  const onChange = useCallback((change: { type: "answer"; operation: "set"; key: string; label: string; value: unknown } | { type: "answer"; operation: "remove"; key: string }) => {
    updateDraft((current) => {
      const answerChanges = { ...current.answerChanges };
      if (change.operation === "remove") delete answerChanges[change.key];
      else answerChanges[change.key] = { label: change.label, value: change.value };
      return { ...current, answerChanges };
    });
  }, [updateDraft]);
  const onPrefill = useCallback((target: string, value: unknown) => {
    updateDraft((current) => withPrefill(ask, current, target, value));
  }, [ask, updateDraft]);

  return (
    <section aria-labelledby="ask-heading" className="mx-auto flex min-h-full w-full max-w-4xl flex-col gap-4">
      <PacketBack returnTo={returnTo} askId={ask.id} />
      <PacketCover
        ask={ask}
        links={{ attachments: ask.attachments, onOpenAttachment: () => undefined, onOpenLinkedArtifact: () => undefined }}
        onOpenAttachment={() => undefined}
        afterQuestion={
          <>
            {(explainer || ask.templateName) && <ExplainerFrame askId={ask.id} kind={ask.kind} recurrence={ask.recurrence} templateName={ask.templateName} templateData={ask.templateData} explainer={explainer} attachments={ask.attachments} onOpen={() => undefined} onPrefill={onPrefill} onAnswerChange={onChange} />}
            {signedIn ? (
              <StatusAnswerBar ask={ask} draft={draft} onDraftChange={updateDraft} onRefresh={refresh} />
            ) : <p role="status" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm">Sign in to send steering or mark this update as read.</p>}
          </>
        }
      />
      <p className="text-xs text-app-muted-foreground">{t(strings.packet.status.requestId)} <bdi dir="ltr" className="select-all font-mono">{ask.id}</bdi></p>
    </section>
  );
}
