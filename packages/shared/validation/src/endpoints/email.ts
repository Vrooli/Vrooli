import { blankToUndefined, req, YupModel } from '../utils';
import * as yup from 'yup';

const emailAddress = yup.string().transform(blankToUndefined).email()

export const emailValidation: YupModel = {
    create: () => yup.object().shape({
        emailAddress: req(emailAddress),
    }),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
}
