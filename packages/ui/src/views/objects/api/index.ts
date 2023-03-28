import { ApiVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { ApiVersionShape } from 'utils/shape/models/apiVersion';

export * from './ApiCreate/ApiCreate';
export * from './ApiUpdate/ApiUpdate';
export * from './ApiView/ApiView';

export const apiInitialValues = (
    session: Session | undefined,
    existing?: ApiVersion | null | undefined
): ApiVersionShape => ({
    __typename: 'ApiVersion' as const,
    id: DUMMY_ID,
    callLink: '',
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Api' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        details: '',
        name: '',
        summary: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});