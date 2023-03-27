import { NoteVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { NoteVersionShape } from 'utils/shape/models/noteVersion';
import { OwnerShape } from 'utils/shape/models/types';

export * from './NoteCreate/NoteCreate';
export * from './NoteUpdate/NoteUpdate';
export * from './NoteView/NoteView';

export const noteInitialValues = (
    session: Session | undefined,
    existing?: NoteVersion | undefined
): NoteVersionShape => ({
    __typename: 'NoteVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isPrivate: true,
    root: {
        id: DUMMY_ID,
        isPrivate: true,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! } as OwnerShape,
        parent: null,
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        name: '',
        text: '',
    }],
    versionLabel: existing?.versionLabel ?? '1.0.0',
    ...existing,
});