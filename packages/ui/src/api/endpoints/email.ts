import { emailPartial, successPartial } from '../partial';
import { toMutation } from '../utils';

export const emailEndpoint = {
    create: toMutation('emailCreate', 'PhoneCreateInput', emailPartial, 'full'),
    verify: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', successPartial, 'full')
}