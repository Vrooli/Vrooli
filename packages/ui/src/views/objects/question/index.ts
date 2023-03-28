import { Question, QuestionForType, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getUserLanguages } from 'utils/display/translationTools';
import { QuestionShape } from 'utils/shape/models/question';

export * from './QuestionCreate/QuestionCreate';
export * from './QuestionUpdate/QuestionUpdate';
export * from './QuestionView/QuestionView';

export const questionInitialValues = (
    session: Session | undefined,
    existing?: Question | null | undefined
): QuestionShape => ({
    __typename: 'Question' as const,
    isPrivate: false,
    referencing: null,
    forType: QuestionForType,
    forConnect: DUMMY_ID,
    tags: [],
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        name: '',
    }],
    ...existing,
});