import { phoneFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const phoneEndpoint = {
    create: toMutation('phoneCreate', 'PhoneCreateInput', fullFields[1]),
    update: toMutation('sendVerificationText', 'SendVerificationTextInput', `{ success }`)
}