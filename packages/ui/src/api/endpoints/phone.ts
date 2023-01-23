import { phonePartial, successPartial } from 'api/partial';
import { toMutation } from 'api/utils';

export const phoneEndpoint = {
    create: toMutation('phoneCreate', 'PhoneCreateInput', phonePartial, 'full'),
    update: toMutation('sendVerificationText', 'SendVerificationTextInput', successPartial, 'full')
}