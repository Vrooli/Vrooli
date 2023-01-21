import { phonePartial, successPartial } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const phoneEndpoint = {
    create: toMutation('phoneCreate', 'PhoneCreateInput', phonePartial, 'full'),
    update: toMutation('sendVerificationText', 'SendVerificationTextInput', successPartial, 'full')
}