import { Session, StandardVersion } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { StandardVersionShape } from 'utils/shape/models/standardVersion';

export * from './StandardCreate/StandardCreate';
export * from './StandardUpdate/StandardUpdate';
export * from './StandardView/StandardView';

export const standardInitialValues = (
    session: Session | undefined,
    existing?: StandardVersion | undefined
): StandardVersionShape => ({
    __typename: 'StandardVersion' as const,
    id: DUMMY_ID,
    default: '',
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    isFile: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Standard' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        jsonVariable: null, //TODO
        name: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});