import { emailPartial, successPartial } from 'api/partial';
import { toMutation } from 'api/utils';

export const emailEndpoint = {
    create: toMutation('emailCreate', 'PhoneCreateInput', emailPartial, 'full'),
    verify: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', successPartial, 'full')
}