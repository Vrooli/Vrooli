import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { FileQuestion } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { resolveLinkedArtifact, type LinkedArtifact } from "../../api/attachments";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { useClaimNavSection } from "../../layout/navSection";
import { useNow } from "../../hooks/useNow";
import { useTranslation } from "../../i18n";
import { SignInForm } from "../session/SessionGate";
import { useSession } from "../session/useSession";
import { ContextRequestSheet } from "./context/ContextRequestSheet";
import { ContextRequests } from "./context/ContextRequests";
import { ExplainerFrame } from "./explainer/ExplainerFrame";
import { ActionBar } from "./form/ActionBar";
import { PinnedBar } from "./form/PinnedBar";
import { SecondaryActions } from "./form/SecondaryActions";
import { AnswerPanel } from "./form/AnswerPanel";
import { withPrefill } from "./model/answer";
import { usePacketDraft } from "./usePacketDraft";
import { PacketBack } from "./PacketBack";
import { PacketCover } from "./PacketCover";
import { Revisions } from "./Revisions";
import { OutcomeNotices } from "./status/OutcomeNotices";
import { PacketStatus } from "./status/PacketStatus";
import { useAnswerFlow } from "./useAnswerFlow";
import { usePacketOrigin } from "./usePacketOrigin";
import { AttachmentStrip } from "./viewer/AttachmentStrip";
import { explainerOf, type AttachmentTarget } from "./viewer/attachments";

// The shared file viewer brings PDF, 3D, syntax and document engines; they load when a file is first opened.
const AttachmentViewer = lazy(() => import("./viewer/AttachmentViewer").then((module) => ({ default: module.AttachmentViewer })));

/** Below the medium breakpoint: the phone layout, where several questions become a guided flow. */
const PHONE = "(max-width: 47.999rem)";

/**
 * One decision, built for a phone: a push opens this page, option selection
 * stays a draft, and the recipient deliberately submits the answer. iOS
 * home-screen apps show no notification action buttons, so this page is the
 * answer path there.
 */
export function PacketPage() {
  const { askId = "" } = useParams();
  const session = useSession();
  const subject = session.data?.subject ?? "";
  const [expiredAttemptSubject, setExpiredAttemptSubject] = useState("");
  useEffect(() => {
    const onExpired = () => {
      if (subject) setExpiredAttemptSubject(subject);
    };
    window.addEventListener("notification-hub:session-expired", onExpired);
    return () => window.removeEventListener("notification-hub:session-expired", onExpired);
  }, [subject]);
  const ask = useQuery({
    queryKey: ["ask", askId, subject],
    enabled: !session.isPending,
    queryFn: async () => (await conversationsClient.getAsk({ askId })).ask,
    retry: (count, error) => !(error instanceof ConnectError && error.code === Code.NotFound) && count < 2,
  });

  if (session.isPending || ask.isPending) return <SkeletonRows rows={3} />;
  if (ask.error instanceof ConnectError && ask.error.code === Code.Unauthenticated) return <SignInForm />;
  if (!ask.data) return <PacketUnavailable error={ask.error} retrying={ask.isFetching} onRetry={() => void ask.refetch()} />;
  return (
    <PacketBody
      key={`${askId}\u0000${subject}`}
      ask={ask.data}
      subject={subject}
      attemptSubject={subject || expiredAttemptSubject}
      signedIn={Boolean(session.data?.signedIn)}
    />
  );
}

/**
 * A decision that could not be shown. Only a refusal from the Hub means it is
 * gone; any other failure may pass, so it can be retried.
 */
function PacketUnavailable({ error, retrying, onRetry }: { error: unknown; retrying: boolean; onRetry: () => void }) {
  const { t } = useTranslation();
  const origin = usePacketOrigin();
  useClaimNavSection(origin.section);
  const gone = error instanceof ConnectError && error.code === Code.NotFound;
  return (
    <section aria-label={t(gone ? strings.packet.status.notFound : strings.packet.status.loadFailed)} className="mx-auto flex w-full max-w-3xl flex-col gap-4">
      <PacketBack returnTo={origin.returnTo} />
      {gone ? (
        <EmptyNotice icon={<FileQuestion size={28} />} title={t(strings.packet.status.notFound)} />
      ) : (
        <QueryError
          error={error}
          title={t(strings.packet.status.loadFailed)}
          detail={t(strings.packet.status.loadFailedDetail)}
          onRetry={onRetry}
          retrying={retrying}
          retryTestId={selectors.packet.loadRetry}
        />
      )}
    </section>
  );
}

interface PacketBodyProps {
  ask: Ask;
  subject: string;
  attemptSubject: string;
  signedIn: boolean;
}

function PacketBody({ ask, subject, attemptSubject, signedIn }: PacketBodyProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { returnTo, section } = usePacketOrigin(ask);
  useClaimNavSection(section);
  const phone = useMediaQuery(PHONE);
  const now = useNow();
  const [draft, updateDraft] = usePacketDraft(subject, ask.id);
  const flow = useAnswerFlow({ ask, subject, attemptSubject, draft });
  const [viewing, setViewing] = useState<AttachmentTarget | null>(null);
  const [linkedArtifact, setLinkedArtifact] = useState<LinkedArtifact>();
  const [linkedArtifactError, setLinkedArtifactError] = useState(false);
  const [resolvingArtifact, setResolvingArtifact] = useState(false);
  const [failedArtifactPath, setFailedArtifactPath] = useState("");
  const [askingContext, setAskingContext] = useState(false);
  const [revisionsOpen, setRevisionsOpen] = useState(false);
  const [answering, setAnswering] = useState(false);
  const [prefilled, setPrefilled] = useState(false);

  const open = ask.state === "pending" || ask.state === "escalated";
  const canAnswer = open || ask.state === "defaulted";
  const explainer = explainerOf(ask.attachments);
  const hasPacket = ask.attachments.length > 0 || ask.questions.length > 0;
  // A sent answer is shown as one stacked, read-only form, never as steps.
  const paged = phone && ask.questions.length >= 2 && !flow.settling;
  const openAttachment = useCallback((target: AttachmentTarget) => {
    setLinkedArtifact(undefined);
    setLinkedArtifactError(false);
    setViewing(target);
  }, []);
  const openLinkedArtifact = useCallback((path: string) => {
    setFailedArtifactPath(path);
    setResolvingArtifact(true);
    setLinkedArtifactError(false);
    void resolveLinkedArtifact(ask.id, path).then((artifact) => {
      setLinkedArtifact(artifact);
      setViewing({ name: "linked-artifact", path: artifact.name });
    }).catch(() => setLinkedArtifactError(true)).finally(() => setResolvingArtifact(false));
  }, [ask.id]);
  const links = useMemo(
    () => ({ attachments: ask.attachments, onOpenAttachment: openAttachment, onOpenLinkedArtifact: openLinkedArtifact }),
    [ask.attachments, openAttachment, openLinkedArtifact],
  );

  const onExplainerPrefill = useCallback(
    (target: string, value: unknown) => {
      updateDraft((current) => {
        const next = withPrefill(ask, current, target, value);
        if (next !== current) setPrefilled(true);
        return next;
      });
    },
    [ask, updateDraft],
  );
  const onExplainerOpen = useCallback((name: string, path: string) => setViewing({ name, path }), []);

  const notedAttachments = useMemo(() => new Set(draft.attachmentNotes.map((item) => item.attachment)), [draft.attachmentNotes]);
  const refreshAsk = () => queryClient.invalidateQueries({ queryKey: ["ask", ask.id, subject] });

  // The cover and the form are two screens only for a guided flow on a phone.
  const showCover = !paged || !answering || !canAnswer || !signedIn;
  const showForm = canAnswer && signedIn && (!paged || answering);
  const form = (
    <AnswerPanel
      ask={ask}
      draft={draft}
      onDraftChange={updateDraft}
      locked={flow.busy || flow.settling}
      frozen={flow.settling}
      paged={paged}
      open={open}
      dismissing={flow.dismissing}
      onSubmit={flow.submit}
      onDismiss={flow.dismiss}
      onRequestContext={() => setAskingContext(true)}
      onOpenAttachment={openAttachment}
      onBackToCover={() => setAnswering(false)}
    />
  );
  const checkStatus = (
    <Button className="w-full" size="lg" type="button" variant="outline" disabled={flow.checking} onClick={flow.checkStatus}>
      {flow.checking ? t(strings.packet.outcome.checking) : t(strings.packet.outcome.checkStatus)}
    </Button>
  );
  const settled = flow.outcome === "recorded" || flow.outcome === "dismissed";
  const hasAnswerColumn = canAnswer && (!signedIn || showForm || (flow.settling && !settled));
  // How a closed decision ended leads the page; a waiting one only mentions its default, further down.
  const status = <PacketStatus ask={ask} now={now} links={links} />;
  const prefillNotice = prefilled && (
    <p role="status" className="rounded-panel border border-app-primary/60 bg-app-primary/10 p-3 text-sm">
      {t(strings.packet.explainer.prefilled)}
    </p>
  );

  return (
    <section aria-labelledby="ask-heading" className={`mx-auto flex min-h-full w-full flex-col gap-3 md:gap-5 ${hasPacket ? "max-w-6xl" : "max-w-3xl"}`}>
      <PacketBack returnTo={returnTo} askId={ask.id} />
      <div className={hasPacket ? "flex flex-1 flex-col gap-4 md:gap-6 lg:grid lg:flex-none lg:grid-cols-[minmax(0,1fr)_26rem] lg:items-start lg:gap-8" : "flex flex-1 flex-col gap-4 md:gap-5"}>
        {showCover ? (
          <div className="flex min-w-0 flex-col gap-4 md:gap-5">
            <PacketCover
              ask={ask}
              links={links}
              onOpenAttachment={openAttachment}
              outcome={open ? undefined : status}
              afterQuestion={
                <>
                  <ContextRequests
                    ask={ask}
                    links={links}
                    onSeeChanges={() => {
                      setRevisionsOpen(true);
                      document.getElementById("packet-revisions")?.scrollIntoView({ behavior: "smooth", block: "start" });
                    }}
                  />
                  {explainer && <ExplainerFrame askId={ask.id} explainer={explainer} attachments={ask.attachments} onOpen={onExplainerOpen} onPrefill={onExplainerPrefill} />}
                </>
              }
            />
            <AttachmentStrip askId={ask.id} attachments={ask.attachments} notedAttachments={notedAttachments} onOpen={openAttachment} />
            <Revisions ask={ask} links={links} open={revisionsOpen} onToggle={setRevisionsOpen} />
            {open && status}
            {resolvingArtifact && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.text.fileLinksLoading)}</p>}
            {linkedArtifactError && (
              <div role="alert" className="flex flex-wrap items-center gap-3 text-sm text-app-muted-foreground">
                <p>{t(strings.packet.text.fileLinksUnavailable)}</p>
                <Button data-testid={selectors.asks.retryFileLinks} className="min-h-11" type="button" variant="outline" disabled={resolvingArtifact} onClick={() => failedArtifactPath && openLinkedArtifact(failedArtifactPath)}>{t(strings.packet.text.retryFileLinks)}</Button>
              </div>
            )}
            <OutcomeNotices
              ask={ask}
              outcome={flow.outcome}
              recordedLabel={flow.recordedLabel}
              attemptedAnswer={flow.attempt?.answer ?? ""}
              sending={flow.sending}
              checking={flow.sent && flow.checking}
              readError={flow.readError}
              returnTo={returnTo}
            />
            <p className="text-xs text-app-muted-foreground">
              {t(strings.packet.status.requestId)} <bdi dir="ltr" className="select-all font-mono">{ask.id}</bdi>
            </p>
          </div>
        ) : null}
        {hasAnswerColumn && (
          // Below the wide layout the column dissolves into the page, so the pinned
          // bar inside it sticks within the whole page; beside the cover it is a
          // panel that stays in view and scrolls on its own.
          <div className="contents lg:flex lg:max-h-[calc(100dvh-2rem)] lg:min-w-0 lg:flex-col lg:gap-4 lg:overflow-y-auto lg:sticky lg:top-4">
            {!signedIn && (
              <div className="flex flex-col gap-3">
                <SignInForm />
                {flow.attempt && flow.outcome !== "recorded" && checkStatus}
              </div>
            )}
            {showForm && prefillNotice}
            {showForm && form}
            {signedIn && flow.settling && !settled && (
              <PinnedBar>{checkStatus}</PinnedBar>
            )}
          </div>
        )}
      </div>
      {canAnswer && signedIn && paged && !answering && (
        <>
          {open && <SecondaryActions canRequestContext={ask.canRequestContext} onRequestContext={() => setAskingContext(true)} onDismiss={flow.dismiss} dismissing={flow.dismissing} disabled={flow.busy} />}
          <ActionBar
            primary={{ label: t(strings.packet.cover.answerQuestions, { count: ask.questions.length }), onClick: () => setAnswering(true), testId: selectors.packet.startAnswer }}
            disabled={flow.busy}
          />
        </>
      )}
      {viewing && (
        <Suspense fallback={null}>
          <div data-testid={selectors.packet.viewer} className="contents">
            <AttachmentViewer
              askId={ask.id}
              attachments={ask.attachments}
              linkedArtifact={linkedArtifact}
              target={viewing}
              onClose={() => { setViewing(null); setLinkedArtifact(undefined); }}
              draft={draft}
              onDraftChange={updateDraft}
              canNote={canAnswer && signedIn && !flow.settling}
            />
          </div>
        </Suspense>
      )}
      {askingContext && (
        <ContextRequestSheet
          askId={ask.id}
          onClose={() => setAskingContext(false)}
          onRefused={() => {
            // The sender stopped accepting questions (a revision can flip it): read the ask again, which drops the button.
            setAskingContext(false);
            void refreshAsk();
          }}
          onSent={() => {
            setAskingContext(false);
            void refreshAsk();
            void queryClient.invalidateQueries({ queryKey: ["asks"] });
          }}
        />
      )}
    </section>
  );
}
