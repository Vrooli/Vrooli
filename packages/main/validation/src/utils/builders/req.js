import { reqErr } from "../errors";
export const req = (field) => {
    return field.required(reqErr);
};
//# sourceMappingURL=req.js.map