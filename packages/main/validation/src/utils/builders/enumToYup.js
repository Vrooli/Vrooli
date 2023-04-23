import * as yup from "yup";
import { blankToUndefined } from "./blankToUndefined";
export const enumToYup = (enumObj) => yup.string().transform(blankToUndefined).oneOf(Object.values(enumObj));
//# sourceMappingURL=enumToYup.js.map