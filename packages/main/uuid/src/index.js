import { NIL, v4 as uuidv4 } from "uuid";
const validateRegex = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;
export const uuid = () => uuidv4();
export const uuidValidate = (uuid) => {
    if (!uuid || typeof uuid !== "string")
        return false;
    return validateRegex.test(uuid);
};
export const DUMMY_ID = NIL;
//# sourceMappingURL=index.js.map