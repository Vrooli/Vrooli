import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { useClaimNavSection } from "../../../layout/navSection";
import { usePacketDraft } from "../usePacketDraft";
import { PacketBack } from "../PacketBack";
import { PacketCover } from "../PacketCover";
import { PacketNavigation, usePacketNavigation, usePacketSwipe } from "../PacketNavigation";
import { withPrefill } from "../model/answer";
import { usePacketOrigin } from "../usePacketOrigin";
import { ExplainerFrame } from "../explainer/ExplainerFrame";
import { StatusAnswerBar } from "./StatusAnswerBar";
import { explainerOf } from "../viewer/attachments";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

export function StatusPacketBody({ ask, subject, signedIn }: { ask: Ask; subject: string; signedIn: boolean }) {
  const { t } = useTranslation();
  const { returnTo, section } = usePacketOrigin(ask);
  const navigation = usePacketNavigation(ask.id, returnTo);
  usePacketSwipe(navigation);
  useClaimNavSection(section);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [draft, updateDraft] = usePacketDraft(subject, ask.id);
  const [interactive, setInteractive] = useState(false);
  const explainer = explainerOf(ask.attachments);
  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["ask", ask.id, subject] });
    void queryClient.invalidateQueries({ queryKey: ["asks"] });
  }, [ask.id, queryClient, subject]);
  const onChange = useCallback((change: { type: "answer"; operation: "set"; key: string; label: string; value: unknown } | { type: "answer"; operation: "remove"; key: string }) => {
    updateDraft((current) => {
      const { [change.key]: _removed, ...remaining } = current.answerChanges;
      const answerChanges = change.operation === "remove"
        ? remaining
        : { ...remaining, [change.key]: { label: change.label, value: change.value } };
      return { ...current, answerChanges };
    });
  }, [updateDraft]);
  const onPrefill = useCallback((target: string, value: unknown) => {
    updateDraft((current) => withPrefill(ask, current, target, value));
  }, [ask, updateDraft]);

  return (
    <section aria-labelledby="ask-heading" className="mx-auto flex min-h-full w-full max-w-6xl flex-col gap-4 pb-24 lg:pb-0">
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
      <div className="flex min-w-0 flex-col gap-4">
      <PacketCover
        ask={ask}
        back={<PacketBack returnTo={returnTo} askId={ask.id} compact />}
        navigation={<PacketNavigation navigation={navigation} />}
        links={{ attachments: ask.attachments, onOpenAttachment: () => undefined, onOpenLinkedArtifact: () => undefined }}
        onOpenAttachment={() => undefined}
        afterQuestion={
          <>
            {(explainer || ask.templateName) && <ExplainerFrame askId={ask.id} subject={subject} kind={ask.kind} recurrence={ask.recurrence} templateName={ask.templateName} templateData={ask.templateData} answerChanges={draft.answerChanges} onAnswerCapability={setInteractive} explainer={explainer} attachments={ask.attachments} onOpen={() => undefined} onOpenAsk={(id) => navigate(`/asks/${encodeURIComponent(id)}`)} onPrefill={onPrefill} onAnswerChange={onChange} />}
          </>
        }
      />
      </div>
      {signedIn ? (
        <StatusAnswerBar ask={ask} draft={draft} onDraftChange={updateDraft} onRefresh={refresh} navigation={navigation} interactive={interactive} />
      ) : <p role="status" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm">{t(strings.packet.statusBar.signedOut)}</p>}
      </div>
    </section>
  );
}
