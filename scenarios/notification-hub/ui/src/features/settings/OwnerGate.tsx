import { useRef, type ReactNode } from "react";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SkeletonRows } from "../../components/Skeleton";
import { SignedOutCard } from "../session/SignedOutCard";
import { SessionUnavailable, SignInForm } from "../session/SessionGate";
import { useSession } from "../session/useSession";
import { Notice } from "./Notice";

/**
 * Shows owner-only content to the signed-in owner and the sign-in card to
 * anyone else, in place of the content.
 *
 * If the session ends while the content is open, the content stays mounted but
 * hidden: an unsaved draft survives a sign-in as the same owner. A different
 * owner gets a fresh page and the previous owner's cached reads are dropped, so
 * one owner's draft and data are never shown to another.
 */
export function OwnerGate({ children, readOnly }: { children: ReactNode; readOnly?: ReactNode }) {
  const { t } = useTranslation();
  const session = useSession();
  const lastOwner = useRef<string | null>(null);
  if (session.isPending) return <SkeletonRows rows={3} />;
  if (session.isError) return <SessionUnavailable retry={() => { void session.refetch(); }} />;

  const signedIn = session.data.signedIn;
  const subject = session.data.subject;
  if (signedIn) {
    lastOwner.current = subject;
  }
  if (lastOwner.current === null) return readOnly ? (
    <div className="flex flex-col gap-4">{readOnly}<SignInForm /></div>
  ) : <SignedOutCard />;

  // The content keeps one position in the tree across expiry and sign-in, so
  // its state is kept; only a change of owner changes its key.
  return (
    <>
      {!signedIn && (
        <div className="flex flex-col gap-3">
          <Notice tone="warning" title={t(strings.settings.sessionEnded.title)}>{t(strings.settings.sessionEnded.body)}</Notice>
          {readOnly ?? <SignedOutCard />}
          {readOnly && <SignInForm />}
        </div>
      )}
      <div hidden={!signedIn} key={lastOwner.current} className={signedIn ? "flex flex-col gap-4" : "hidden"}>{children}</div>
    </>
  );
}
