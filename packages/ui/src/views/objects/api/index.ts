import { ApiVersion, Session } from '@shared/consts';
import { uuid } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { ApiVersionShape } from 'utils/shape/models/apiVersion';

export * from './ApiCreate/ApiCreate';
export * from './ApiUpdate/ApiUpdate';
export * from './ApiView/ApiView';

export const apiInitialValues = (
    session: Session | undefined,
    existing?: ApiVersion | undefined
): ApiVersionShape => ({
    __typename: 'ApiVersion' as const,
    id: uuid(),
    callLink: '',
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: uuid(),
    },
    root: {
        __typename: 'Api' as const,
        id: uuid(),
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
    },
    translations: [{
        id: uuid(),
        language: getUserLanguages(session)[0],
        details: '',
        name: '',
        summary: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});