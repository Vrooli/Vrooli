import { Organization, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { organizationValidation } from '@shared/validation';
import { getUserLanguages } from 'utils/display/translationTools';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { OrganizationShape, shapeOrganization } from 'utils/shape/models/organization';

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

export const transformOrganizationValues = (o: OrganizationShape, u?: OrganizationShape) => {
    return u === undefined
        ? shapeOrganization.create(o)
        : shapeOrganization.update(o, u)
}

export const validateOrganizationValues = async (values: OrganizationShape, isCreate: boolean) => {
    const transformedValues = transformOrganizationValues(values);
    const validationSchema = isCreate
        ? organizationValidation.create({})
        : organizationValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}