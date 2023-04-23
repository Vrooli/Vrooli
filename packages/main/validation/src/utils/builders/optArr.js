import * as yup from "yup";
import { opt } from "./opt";
export const optArr = (field) => {
    return yup.array().of(opt(field));
};
//# sourceMappingURL=optArr.js.map