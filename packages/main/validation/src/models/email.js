import { blankToUndefined, req, yupObj } from "../utils";
import * as yup from "yup";
const emailAddress = yup.string().transform(blankToUndefined).email();
export const emailValidation = {
    create: ({ o }) => yupObj({
        emailAddress: req(emailAddress),
    }, [], [], o),
};
//# sourceMappingURL=email.js.map