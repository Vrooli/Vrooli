import { useRef, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SkeletonRows } from "../../components/Skeleton";
import { SignedOutCard } from "../session/SignedOutCard";
import { useSession } from "../session/useSession";
import { Notice } from "./Notice";
import { OWNER_QUERY_ROOTS } from "./ownerQueries";

/**
 * Shows owner-only content to the signed-in owner and the sign-in card to
 * anyone else, in place of the content.
 *
 * If the session ends while the content is open, the content stays mounted but
 * hidden: an unsaved draft survives a sign-in as the same owner. A different
 * owner gets a fresh page and the previous owner's cached reads are dropped, so
 * one owner's draft and data are never shown to another.
 */
export function OwnerGate({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const session = useSession();
  const lastOwner = useRef<string | null>(null);
  if (session.isPending) return <SkeletonRows rows={3} />;

  const signedIn = Boolean(session.data?.signedIn);
  const subject = session.data?.subject ?? "";
  if (signedIn) {
    if (lastOwner.current !== null && lastOwner.current !== subject) {
      for (const root of OWNER_QUERY_ROOTS) queryClient.removeQueries({ queryKey: [root] });
    }
    lastOwner.current = subject;
  }
  if (lastOwner.current === null) return <SignedOutCard />;

  // The content keeps one position in the tree across expiry and sign-in, so
  // its state is kept; only a change of owner changes its key.
  return (
    <>
      {!signedIn && (
        <div className="flex flex-col gap-3">
          <Notice tone="warning" title={t(strings.settings.sessionEnded.title)}>{t(strings.settings.sessionEnded.body)}</Notice>
          <SignedOutCard />
        </div>
      )}
      <div hidden={!signedIn} key={lastOwner.current} className={signedIn ? "flex flex-col gap-4" : "hidden"}>{children}</div>
    </>
  );
}
