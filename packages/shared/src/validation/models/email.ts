import * as yup from "yup";
import { req, YupModel, yupObj } from "../utils";

const emailAddress = yup.string().trim().removeEmptyString().email();

export const emailValidation: YupModel<true, false> = {
    create: (d) => yupObj({
        emailAddress: req(emailAddress),
    }, [], [], d),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
};
