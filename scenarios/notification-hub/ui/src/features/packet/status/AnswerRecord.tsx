import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { DecisionText } from "../DecisionText";
import type { TextLinks } from "../decisionLinks";
import { describeResponse } from "../model/describeResponse";
import { answerFileUrl } from "../../../api/attachments";

/**
 * What the operator said besides the choice: what they said to each typed
 * question, and the notes pinned to files. In their own words and the
 * question's own labels; the requester's raw option keys never show.
 */
export function AnswerRecord({ ask, links }: { ask: Ask; links: TextLinks }) {
  const { t } = useTranslation();
  const said = ask.responses.flatMap((response) => {
    const question = ask.questions.find((item) => item.id === response.questionId);
    const lines = question ? describeResponse(question, response, t) : [];
    return question && lines.length > 0 ? [{ question, lines }] : [];
  });
  if (said.length === 0 && ask.attachmentNotes.length === 0 && ask.answerAttachments.length === 0) return null;
  return (
    <dl className="flex flex-col gap-2">
      {said.map(({ question, lines }) => (
        <div key={question.id}>
          <dt dir="auto" className="break-words text-app-muted-foreground">{question.prompt}</dt>
          <dd className="flex flex-col">
            {lines.map((line) => <DecisionText key={line} value={line} links={links} inheritText className="text-sm" />)}
          </dd>
        </div>
      ))}
      {ask.attachmentNotes.length > 0 && (
        <div>
          <dt className="text-app-muted-foreground">{t(strings.packet.status.answerFileNotes)}</dt>
          <dd className="flex flex-col gap-1">
            {ask.attachmentNotes.map((note) => (
              <p key={`${note.attachment}\u0000${note.path}`}>
                <bdi dir="ltr" className="font-mono text-xs">{note.path ? `${note.attachment}/${note.path}` : note.attachment}</bdi>
                {": "}
                <DecisionText value={note.note} links={links} inheritText />
              </p>
            ))}
          </dd>
        </div>
      )}
      {ask.answerAttachments.length > 0 && (
        <div>
          <dt className="text-app-muted-foreground">{t(strings.packet.status.answerFileNotes)}</dt>
          <dd className="flex flex-col gap-1">
            {ask.answerAttachments.map((file) => <a key={file.name} href={answerFileUrl(ask.id, file.name)} className="inline-flex min-h-11 items-center break-all underline">{file.name}</a>)}
          </dd>
        </div>
      )}
    </dl>
  );
}
