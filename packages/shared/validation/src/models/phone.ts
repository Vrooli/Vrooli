import { phoneNumber, req, YupModel, yupObj } from "../utils";

export const phoneValidation: YupModel<true, false> = {
    create: ({ o }) => yupObj({
        phoneNumber: req(phoneNumber),
    }, [], [], o),
    // Can't update an phone. Push notifications & other phone-related settings are updated elsewhere
}
