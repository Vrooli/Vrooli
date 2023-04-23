import { CustomError } from "../events";
import { ObjectMap } from "../models";
export function getLogic(props, objectType, languages, errorTrace) {
    const object = ObjectMap[objectType];
    if (!object)
        throw new CustomError("0280", "InvalidArgs", languages, { errorTrace, objectType });
    if (!props.length)
        return object;
    for (const field of props) {
        const logic = object[field];
        if (!logic)
            throw new CustomError("0367", "InvalidArgs", languages, { errorTrace, objectType, field });
    }
    return object;
}
//# sourceMappingURL=getLogic.js.map