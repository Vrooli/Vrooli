import { emailFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const emailEndpoint = {
    create: toMutation('emailCreate', 'PhoneCreateInput', [fullFields], `...fullFields`),
    update: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', [], `success`)
}