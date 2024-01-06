import { phoneNumber, req, YupModel, yupObj } from "../utils";

export const phoneValidation: YupModel<["create"]> = {
    create: (d) => yupObj({
        phoneNumber: req(phoneNumber),
    }, [], [], d),
    // Can't update an phone. Push notifications & other phone-related settings are updated elsewhere
};
