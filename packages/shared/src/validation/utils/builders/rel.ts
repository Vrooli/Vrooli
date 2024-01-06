import { OrArray, YupModelOptions } from "@local/shared";
import * as yup from "yup";
import { ObjectShape } from "yup/lib/object";
import { id } from "../commonFields";
import { YupModel, YupMutateParams } from "../types";
import { opt } from "./opt";
import { optArr } from "./optArr";
import { req } from "./req";
import { reqArr } from "./reqArr";

export type RelationshipType = "Connect" | "Create" | "Delete" | "Disconnect" | "Update";

type RelOutput<FieldName extends string> = (
    ({ [x in `${FieldName}Connect`]?: OrArray<string> }) &
    ({ [x in `${FieldName}Create`]?: OrArray<yup.ObjectSchema<ObjectShape>> }) &
    ({ [x in `${FieldName}Delete`]?: number | boolean }) &
    ({ [x in `${FieldName}Disconnect`]?: number | boolean }) &
    ({ [x in `${FieldName}Update`]?: OrArray<yup.ObjectSchema<ObjectShape>> })
)

/**
 * Creates the validation fields for a relationship
 * @param data Parameters for YupModel create and update
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param isRequired "opt" or "req" to mark the fields as optional or required. 
 * If required, only one "Connect" or "Create" is actually marked as required, and the other 
 * fields are marked as optional. If optional, all fields are marked as optional.
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @param model The relationship's validation object
 * @param omitRels An array of fields to omit from the validation object. Replaces data.omitRels
 * @returns An object with the validation fields for the relationship
 */
export const rel = <
    RelTypes extends readonly RelationshipType[],
    FieldName extends string,
    // Model only required when RelTypes includes 'Create' or 'Update'
    Model extends YupModel<YupModelOptions[]>,
    OmitField extends string,
>(
    data: YupMutateParams,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: "one" | "many",
    isRequired: "opt" | "req",
    // Model only required if relTypes includes 'Create' or 'Update'
    model?: Model,
    omitRels?: OmitField | OmitField[],
): RelOutput<FieldName> => {
    // Check if model is required
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
        if (!model) throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    // Initialize result
    const result: RelOutput<FieldName> = {};
    // Loop through relation types
    for (const t of relTypes) {
        // Determine if field is required. If both 'Connect' and 'Create' are allowed, both 
        // are marked as optional here. This is because yup defines this one-of rule in the second
        // parameter of object.shape()
        // Count the number of times 'Connect' and 'Create' are in relTypes
        const connectCreateCount = relTypes.filter(x => x === "Connect" || x === "Create").length;
        const required = isRequired === "req" && connectCreateCount === 1 && (t === "Connect" || t === "Create");
        // Add validation field to result
        if (t === "Connect") {
            result[`${relation}${t}`] = isOneToOne === "one" ?
                required ? req(id) : opt(id) :
                required ? reqArr(id) : optArr(id);
        }
        else if (t === "Create") {
            result[`${relation}${t}`] = isOneToOne === "one" ?
                required ? req((model as YupModel<["create", "update"]>).create({ ...data, omitRels })) : opt((model as YupModel<["create", "update"]>).create({ ...data, omitRels })) :
                required ? reqArr((model as YupModel<["create", "update"]>).create({ ...data, omitRels })) : optArr((model as YupModel<["create", "update"]>).create({ ...data, omitRels }));
        }
        else if (t === "Delete") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool()) : optArr(id);
        }
        else if (t === "Disconnect") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool()) : optArr(id);
        }
        else if (t === "Update") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt((model as YupModel<["create", "update"]>).update({ ...data, omitRels })) : optArr((model as YupModel<["create", "update"]>).update({ ...data, omitRels }));
        }
    }
    return result;
};
