import { useTranslation } from "react-i18next";
import { ArrowLeft, KeyRound, ShieldAlert } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import type { JoinRequest } from "../../api/machines";
import { humanAge } from "./age";
import { IconButton } from "@vrooli/react-component-library/IconButton";

/**
 * Screen 03 — confirming it is the right machine.
 *
 * Every descriptive field in a join request is chosen by whoever sent it, so a
 * name proves nothing: on a shared network there is otherwise no way to tell
 * your laptop from something else claiming to be it. The three words are
 * derived from both machines' public keys, so matching words mean matching
 * keys. This is the handshake SSH, Signal and Tailscale all settled on.
 */

interface ReviewRequestProps {
  request: JoinRequest;
  onBack: () => void;
  onDeny: () => void;
  onContinue: () => void;
  denying: boolean;
}

export default function ReviewRequest({ request, onBack, onDeny, onContinue, denying }: ReviewRequestProps) {
  const { t } = useTranslation();
  const meta = [request.os, request.arch, request.endpoint, t(strings.machines.requestedAgo, { age: humanAge(request.requestedAgeSeconds) })]
    .filter(Boolean)
    .join(" · ");
  // Without derived words there is nothing to compare, so the safe action is
  // the only one offered.
  const hasWords = request.confirmationWords.length > 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mx-auto flex w-full max-w-2xl items-center gap-2 px-5 pt-5">
        <IconButton
          data-testid="machines-review-back"
          onClick={onBack}
          aria-label={t(strings.machines.back)}
          shape="rounded"
        >
          <ArrowLeft aria-hidden />
        </IconButton>
        <div className="min-w-0">
          <h2 className="truncate text-lg font-semibold text-wc-text-primary">
            {t(strings.machines.reviewTitle, { name: request.name })}
          </h2>
          <p className="truncate text-xs text-wc-text-faint">{meta}</p>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-2xl space-y-5 px-5 pb-5 pt-5">
        {hasWords ? (
          <section className="rounded-2xl border border-wc-accent/30 bg-wc-accent/5 p-5 text-center">
            <p className="text-sm leading-6 text-wc-text-muted">
              {t(strings.machines.reviewCheck, { name: request.name })}
            </p>
            <p
              data-testid="machines-confirmation-words"
              className="mt-4 font-mono text-2xl font-medium tracking-[0.08em] text-wc-text-primary sm:text-3xl"
            >
              {request.confirmationWords.join(" · ")}
            </p>
            <p className="mt-3 inline-flex flex-wrap items-center justify-center gap-1.5 text-xs text-wc-text-faint">
              <KeyRound className="h-3.5 w-3.5" aria-hidden />
              {t(strings.machines.reviewDerived)}
              {request.keyFingerprint && (
                <span data-testid="machines-key-fingerprint" className="font-mono">
                  · {request.keyFingerprint}
                </span>
              )}
            </p>
          </section>
        ) : (
          <div
            data-testid="machines-no-words"
            role="alert"
            className="flex gap-3 rounded-xl border border-rose-400/25 bg-rose-400/10 p-4 text-sm leading-6 text-rose-100"
          >
            <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
            <span>{t(strings.machines.noWords)}</span>
          </div>
        )}

        <p className="text-center text-xs leading-5 text-wc-text-faint">{t(strings.machines.reviewMismatch)}</p>
        </div>
      </div>

      <footer className="shrink-0 border-t border-wc-default px-5 py-3">
        <div className="mx-auto flex w-full max-w-2xl flex-wrap items-center justify-end gap-2">
        <Button variant="outline" data-testid="machines-deny" onClick={onDeny} disabled={denying}>
          {t(strings.machines.deny)}
        </Button>
        <Button data-testid="machines-words-match" onClick={onContinue} disabled={!hasWords || denying}>
          {t(strings.machines.wordsMatch)}
        </Button>
      </div>
      </footer>
    </div>
  );
}
