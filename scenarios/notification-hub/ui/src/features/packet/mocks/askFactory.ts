import { create } from "@bufbuild/protobuf";
import {
  AskAttachmentEntrySchema,
  AskAttachmentSchema,
  AskAttachment_Kind,
  AskAttachment_Role,
  AskOptionSchema,
  AskQuestionChoiceSchema,
  AskQuestionResponseSchema,
  AskQuestionSchema,
  AskQuestionSectionSchema,
  AskSchema,
  type Ask,
  type AskAttachment,
  type AskQuestion,
} from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";


/** A pending decision whose sender answers questions, with options and a recommendation; override what a test is about. */
export function makeAsk(overrides: Partial<Omit<Ask, "$typeName" | "options">> & { options?: { key: string; label: string }[] } = {}): Ask {
  const { options, ...rest } = overrides;
  return create(AskSchema, {
    id: "ask-1",
    question: "Close the BAS goal or re-aim at quality?",
    recommended: "re-aim",
    recommendationReason: "Quality gaps remain.",
    defaultAnswer: "re-aim",
    reversible: true,
    canRequestContext: true,
    state: "pending",
    ...rest,
    options: (options ?? [
      { key: "close", label: "Close the goal" },
      { key: "re-aim", label: "Re-aim at quality" },
      { key: "other", label: "Other — I have a different answer" },
    ]).map((option) => create(AskOptionSchema, option)),
  });
}

type QuestionInit = {
  id: string;
  type: string;
  prompt: string;
  help?: string;
  required?: boolean;
  choices?: { key: string; label: string; detail?: string; attachment?: string }[];
  sections?: { key: string; title: string; summary?: string; attachment?: string }[];
  recommended?: { selected?: string[]; text?: string; sections?: { sectionKey: string; verdict: string; note?: string }[] };
  recommendationReason?: string;
  unansweredVerdict?: string;
};

export function makeQuestion(init: QuestionInit): AskQuestion {
  const { choices = [], sections = [], recommended, ...rest } = init;
  return create(AskQuestionSchema, {
    ...rest,
    choices: choices.map((choice) => create(AskQuestionChoiceSchema, choice)),
    sections: sections.map((section) => create(AskQuestionSectionSchema, section)),
    recommended: recommended ? create(AskQuestionResponseSchema, { questionId: init.id, ...recommended }) : undefined,
  });
}

type AttachmentInit = {
  name: string;
  label?: string;
  folder?: boolean;
  explainer?: boolean;
  sourcePath?: string;
  sourceStatus?: string;
  previewKind?: string;
  mediaType?: string;
  sizeBytes?: number;
  sha256?: string;
  entries?: { path: string; previewKind?: string; mediaType?: string; sizeBytes?: number; sha256?: string }[];
};

export function makeAttachment({ folder, explainer, entries = [], sizeBytes = 1024, ...rest }: AttachmentInit): AskAttachment {
  return create(AskAttachmentSchema, {
    sha256: `sha-${rest.name}`,
    mediaType: "text/markdown",
    previewKind: "markdown",
    ...rest,
    sizeBytes: BigInt(sizeBytes),
    kind: folder ? AskAttachment_Kind.FOLDER : AskAttachment_Kind.FILE,
    role: explainer ? AskAttachment_Role.EXPLAINER : AskAttachment_Role.EVIDENCE,
    entries: entries.map(({ sizeBytes: size = 10, ...entry }) => create(AskAttachmentEntrySchema, { mediaType: "text/plain", previewKind: "text", ...entry, sizeBytes: BigInt(size) })),
  });
}
