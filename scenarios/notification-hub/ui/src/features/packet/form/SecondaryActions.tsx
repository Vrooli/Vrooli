import { Button } from "@vrooli/react-component-library/Button/2";
import { MessageCircleQuestion } from "lucide-react";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

/**
 * What to do when the decision cannot be answered yet: ask the requester for
 * more context (the decision stays open), or dismiss it without answering.
 * They sit in the flow below the form, away from the Send button, so neither
 * can be tapped by mistake. A requester that did not say it answers questions
 * gets no dead button, only a line saying so.
 */
export function SecondaryActions({ canRequestContext, onRequestContext, onDismiss, dismissing, disabled }: { canRequestContext: boolean; onRequestContext: () => void; onDismiss: () => void; dismissing: boolean; disabled: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2 border-t border-app-border pt-4 sm:flex-row">
      {canRequestContext ? (
        <Button data-testid={selectors.packet.needContext} className="flex-1 border border-app-border bg-app-surface" type="button" variant="outline" disabled={disabled} onClick={onRequestContext}>
          <MessageCircleQuestion size={16} aria-hidden="true" className="me-1.5" />
          {t(strings.packet.context.open)}
        </Button>
      ) : (
        <p data-testid={selectors.packet.contextUnavailable} className="flex-1 self-center text-sm text-app-muted-foreground">{t(strings.packet.context.unavailable)}</p>
      )}
      <Button className="flex-1 border border-app-border bg-app-surface" type="button" variant="outline" disabled={disabled} onClick={onDismiss}>
        {t(dismissing ? strings.packet.form.dismissing : strings.packet.form.dismiss)}
      </Button>
    </div>
  );
}
