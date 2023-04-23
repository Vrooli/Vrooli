import { phoneNumber, req, yupObj } from "../utils";
export const phoneValidation = {
    create: ({ o }) => yupObj({
        phoneNumber: req(phoneNumber),
    }, [], [], o),
};
//# sourceMappingURL=phone.js.map