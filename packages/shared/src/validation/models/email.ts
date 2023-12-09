import { email, req, YupModel, yupObj } from "../utils";

export const emailValidation: YupModel<true, false> = {
    create: (d) => yupObj({
        emailAddress: req(email),
    }, [], [], d),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
};
