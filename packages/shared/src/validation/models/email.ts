import * as yup from "yup";
import { blankToUndefined, req, YupModel, yupObj } from "../utils";

const emailAddress = yup.string().transform(blankToUndefined).email();

export const emailValidation: YupModel<true, false> = {
    create: ({ o }) => yupObj({
        emailAddress: req(emailAddress),
    }, [], [], o),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
};
