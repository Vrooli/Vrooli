import * as yup from "yup";
import { YupModelOptions } from "../../../validation/utils/types.js";
import { id, language } from "../commonFields.js";
import { YupModel, YupMutateParams } from "../types.js";
import { opt, optArr, req, reqArr } from "./optionality.js";
import { yupObj } from "./yupObj.js";

export type RelationshipType = "Connect" | "Create" | "Delete" | "Disconnect" | "Update";

type YupString = yup.StringSchema;
type YupStringArray = yup.ArraySchema<string[], yup.AnyObject, "", "">
type YupObject = yup.ObjectSchema<yup.ObjectShape>;
type YupObjectArray = yup.ArraySchema<unknown[], yup.AnyObject, "", "">;
type YupBoolean = yup.BooleanSchema;
type RelOutput<FieldName extends string> = (
    ({ [x in `${FieldName}Connect`]?: YupString | YupStringArray }) &
    ({ [x in `${FieldName}Create`]?: YupObject | YupObjectArray }) &
    ({ [x in `${FieldName}Delete`]?: YupBoolean | YupStringArray }) &
    ({ [x in `${FieldName}Disconnect`]?: YupBoolean | YupStringArray }) &
    ({ [x in `${FieldName}Update`]?: YupObject | YupObjectArray })
)

const REL_RECURSE_LIMIT = 20;

/**
 * Creates the validation fields for a relationship
 * @param data Parameters for YupModel create and update
 * @param recurseCount Used internally to prevent infinite recursion
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param isRequired "opt" or "req" to mark the fields as optional or required. 
 * If required, only one "Connect" or "Create" is actually marked as required, and the other 
 * fields are marked as optional. If optional, all fields are marked as optional.
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @param model The relationship's validation object
 * @param omitFields An array of fields to omit from the validation object. Replaces data.omitFields
 * @returns An object with the validation fields for the relationship
 */
export function rel<
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
    directOmitFields?: OmitField | OmitField[],
): RelOutput<FieldName> {
    // Initialize result
    const result: RelOutput<FieldName> = {};
    if (data.recurseCount && data.recurseCount > REL_RECURSE_LIMIT) {
        console.warn("Hit recursion limit in rel", relation, relTypes, isOneToOne, isRequired, model, directOmitFields);
        return result;
    }
    // Check if model is required
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
        if (!model) throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    // There are two pass to pass omitFields. Combine them into a single array
    const omitFields = Array.from(new Set([
        ...(Array.isArray(data.omitFields) ? data.omitFields : data.omitFields ? [data.omitFields] : []),
        ...(Array.isArray(directOmitFields) ? directOmitFields : directOmitFields ? [directOmitFields] : []),
    ]));
    // Helper function to wrap the field in an optional or required object or array
    function wrap<T extends yup.AnySchema>(field: T, required: boolean): T {
        return isOneToOne === "one" ?
            required ? req(field) : opt(field) :
            required ? reqArr(field) : optArr(field);
    }
    const recurseCount = (data.recurseCount || 0) + 1;
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
            result[`${relation}${t}`] = wrap(id, required) as RelOutput<FieldName>[`${FieldName}Connect`];
        }
        else if (t === "Create") {
            result[`${relation}${t}`] = wrap((model as YupModel<["create", "update"]>).create({ ...data, omitFields, recurseCount }), required) as RelOutput<FieldName>[`${FieldName}Create`];
        }
        else if (t === "Delete") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool().oneOf([true], "Must be true")) : optArr(id);
        }
        else if (t === "Disconnect") {
            result[`${relation}${t}`] = isOneToOne === "one" ? opt(yup.bool().oneOf([true], "Must be true")) : optArr(id);
        }
        else if (t === "Update") {
            result[`${relation}${t}`] = wrap((model as YupModel<["create", "update"]>).update({ ...data, omitFields, recurseCount }), required) as RelOutput<FieldName>[`${FieldName}Update`];
        }
    }
    return result;
}

/**
 * Builds a YupModel for a translation object. 
 * All translation objects function the same way: 
 * - They have an id and language field
 * - They have additional, optional or required fields
 * @param parialYupModel Partial YupModel for the translation object. Only 
 * includes the additional fields
 * @returns YupModel for the translation object
 */
export function transRel(partialYupModel: ({
    create: (params: YupMutateParams) => { [key: string]: yup.StringSchema };
    update: (params: YupMutateParams) => { [key: string]: yup.StringSchema };
})): YupModel<["create", "update"]> {
    return {
        create: (data) => yupObj({
            id: req(id),
            language: req(language),
            ...partialYupModel.create(data),
        }, [], [], data),
        update: (data) => yupObj({
            id: req(id),
            language: opt(language),
            ...partialYupModel.update(data),
        }, [], [], data),
    };
}
