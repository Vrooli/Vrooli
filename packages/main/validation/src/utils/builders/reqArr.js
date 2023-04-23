import * as yup from "yup";
import { req } from "./req";
export const reqArr = (field) => {
    return yup.array().of(req(field));
};
//# sourceMappingURL=reqArr.js.map