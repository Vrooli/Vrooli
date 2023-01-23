import { phonePartial, successPartial } from '../partial';
import { toMutation } from '../utils';

export const phoneEndpoint = {
    create: toMutation('phoneCreate', 'PhoneCreateInput', phonePartial, 'full'),
    update: toMutation('sendVerificationText', 'SendVerificationTextInput', successPartial, 'full')
}