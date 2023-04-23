import * as yup from "yup";
import { id } from "../commonFields";
import { opt } from "./opt";
import { optArr } from "./optArr";
import { req } from "./req";
import { reqArr } from "./reqArr";
export const rel = (relation, relTypes, isOneToOne, isRequired, model, omitFields) => {
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
        if (!model)
            throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    const result = {};
    for (const t of relTypes) {
        const connectCreateCount = relTypes.filter(x => x === "Connect" || x === "Create").length;
        const required = isRequired === "req" && connectCreateCount === 1 && (t === "Connect" || t === "Create");
        if (t === "Connect") {
            result[`${relation}${t}`] = isOneToOne === "one" ?
                required ? req(id) : opt(id) :
                required ? reqArr(id) : optArr(id);
        }
        else if (t === "Create") {
            result[`${relation}${t}`] = isOneToOne === "one" ?
                required ? req(model.create({ o: omitFields })) : opt(model.create({ o: omitFields })) :
                required ? reqArr(model.create({ o: omitFields })) : optArr(model.create({ o: omitFields }));
        }
        else if (t === "Delete") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool()) : optArr(id);
        }
        else if (t === "Disconnect") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool()) : optArr(id);
        }
        else if (t === "Update") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(model.update({ o: omitFields })) : optArr(model.update({ o: omitFields }));
        }
    }
    return result;
};
//# sourceMappingURL=rel.js.map