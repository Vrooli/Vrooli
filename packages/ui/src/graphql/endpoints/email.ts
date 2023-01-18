import { emailFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const emailEndpoint = {
    create: toMutation('emailCreate', 'PhoneCreateInput', fullFields[1]),
    verify: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', `{ success }`)
}