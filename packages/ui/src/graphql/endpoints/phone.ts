import { phoneFields as fullFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const phoneEndpoint = {
    create: toMutation('phoneCreate', 'PhoneCreateInput', [fullFields], `...fullFields`),
    update: toMutation('sendVerificationText', 'SendVerificationTextInput', [], `success`)
}