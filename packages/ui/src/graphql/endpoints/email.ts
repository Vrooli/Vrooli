import { emailFields as fullFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const emailEndpoint = {
    create: toGql('mutation', 'emailCreate', 'EmailCreateInput', [fullFields], `...fullFields`),
    verify: toGql('mutation', 'sendVerificationEmail', 'SendVerificationEmailInput', [], `success`)
}