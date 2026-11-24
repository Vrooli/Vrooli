import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";

import { conversationsClient } from "../../../api/notifications";
import { Dialog } from "@vrooli/react-component-library/Dialog/1.3.10";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { REASON_CONTEXT_NOT_ACCEPTED, hasHubReason } from "../hubError";
import { TextArea } from "../form/TextArea";
import { ANSWER_LIMITS } from "../model/questions";

/**
 * "I need more context": the operator says what is unclear, the decision
 * stays open, and the requester is asked to revise it. When the Hub refuses
 * because the sender does not accept questions, the sheet closes and the page
 * reads the ask again. A bottom sheet on a
 * phone, a dialog on a wide screen.
 */
export function ContextRequestSheet({ askId, onClose, onSent, onRefused }: { askId: string; onClose: () => void; onSent: () => void; onRefused: () => void }) {
  const { t } = useTranslation();
  const [question, setQuestion] = useState("");
  const send = useMutation({
    retry: false,
    mutationFn: () => conversationsClient.requestContext({ askId, question: question.trim() }),
    onSuccess: onSent,
    onError: (error) => {
      if (hasHubReason(error, REASON_CONTEXT_NOT_ACCEPTED)) onRefused();
    },
  });
  const ready = question.trim().length > 0 && !send.isPending;
  return (
    <Dialog
      open
      title={t(strings.packet.context.title)}
      description={t(strings.packet.context.description)}
      closeLabel={t(strings.packet.context.cancel)}
      onClose={onClose}
      footer={
        <div className="flex flex-col gap-2">
          {send.isError && <p role="alert" className="text-sm text-app-danger">{t(strings.packet.context.error)}</p>}
          <div className="flex gap-2">
            <Button type="button" variant="outline" className="min-h-12" disabled={send.isPending} onClick={onClose}>
              {t(strings.packet.context.cancel)}
            </Button>
            <Button data-testid={selectors.packet.sendContext} type="button" variant="primary" className="min-h-12 flex-1" disabled={!ready} onClick={() => send.mutate()}>
              {t(send.isPending ? strings.packet.context.sending : strings.packet.context.send)}
            </Button>
          </div>
        </div>
      }
    >
      <TextArea
        label={t(strings.packet.context.label)}
        testId={selectors.packet.contextQuestion}
        value={question}
        rows={5}
        maxLength={ANSWER_LIMITS.contextQuestionChars}
        placeholder={t(strings.packet.context.placeholder)}
        onChange={setQuestion}
      />
    </Dialog>
  );
}
