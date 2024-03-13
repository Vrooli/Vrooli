import { email, id, req, YupModel, yupObj } from "../utils";

export const emailValidation: YupModel<["create"]> = {
    create: (d) => yupObj({
        id: req(id),
        emailAddress: req(email),
    }, [], [], d),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
};
