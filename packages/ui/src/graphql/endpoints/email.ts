import { emailPartial, successPartial } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const emailEndpoint = {
    create: toMutation('emailCreate', 'PhoneCreateInput', emailPartial, 'full'),
    verify: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', successPartial, 'full')
}