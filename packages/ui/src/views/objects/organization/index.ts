import { Organization, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getUserLanguages } from 'utils/display/translationTools';
import { OrganizationShape } from 'utils/shape/models/organization';

export * from './OrganizationCreate/OrganizationCreate';
export * from './OrganizationUpdate/OrganizationUpdate';
export * from './OrganizationView/OrganizationView';

export const organizationInitialValues = (
    session: Session | undefined,
    existing?: Organization | null | undefined
): OrganizationShape => ({
    __typename: 'Organization' as const,
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: '',
        bio: '',
    }],
    ...existing,
});