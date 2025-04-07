import * as yup from "yup";
import { YupModelOptions } from "../../../validation/utils/types.js";
import { id, language } from "../commonFields.js";
import { YupModel, YupMutateParams } from "../types.js";
import { opt, req, reqArr } from "./optionality.js";
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

    // Skip this relationship entirely if it's in the omitFields list
    if (omitFields.includes(relation)) {
        return result;
    }

    // Helper function to wrap the field in an optional or required object or array
    function wrap<T extends yup.AnySchema>(field: T, required: boolean): T {
        if (isOneToOne === "one") {
            // For one-to-one relationships
            if (required) {
                return req(field);
            } else {
                // Make the field optional, and if undefined, don't validate inner required fields
                return field.notRequired().nullable().default(undefined);
            }
        } else {
            // For one-to-many relationships - don't strip arrays
            if (required) {
                return req(yup.array().of(field)) as unknown as T;
            } else {
                return yup.array().of(field).notRequired().nullable() as unknown as T;
            }
        }
    }

    const recurseCount = (data.recurseCount || 0) + 1;
    // Loop through relation types
    for (const t of relTypes) {
        // Skip this relationship type if it's in the omitFields list (e.g. relationCreate)
        const fieldName = `${relation}${t}`;
        if (omitFields.includes(fieldName)) {
            continue;
        }

        // Determine if field is required. If both 'Connect' and 'Create' are allowed, both 
        // are marked as optional here. This is because yup defines this one-of rule in the second
        // parameter of object.shape()
        // Count the number of times 'Connect' and 'Create' are in relTypes
        const connectCreateCount = relTypes.filter(x => x === "Connect" || x === "Create").length;
        const required = isRequired === "req" && connectCreateCount === 1 && (t === "Connect" || t === "Create");
        // Add validation field to result
        if (t === "Connect") {
            // For Connect fields, we need to make sure they're preserved in the output
            // even when optional (not stripped)
            if (isOneToOne === "one") {
                result[`${relation}${t}`] = required ?
                    req(id) :
                    id.notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Connect`];
            } else {
                result[`${relation}${t}`] = required ?
                    reqArr(id) :
                    yup.array().of(id).notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Connect`];
            }
        }
        else if (t === "Create") {
            result[`${relation}${t}`] = wrap((model as YupModel<["create", "update"]>).create({ ...data, omitFields, recurseCount }), required) as RelOutput<FieldName>[`${FieldName}Create`];
        }
        else if (t === "Delete") {
            // For Delete fields, ensure they're preserved in the output
            if (isOneToOne === "one") {
                result[`${relation}${t}`] = yup.bool().oneOf([true], "Must be true").notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Delete`];
            } else {
                result[`${relation}${t}`] = yup.array().of(id).notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Delete`];
            }
        }
        else if (t === "Disconnect") {
            // For Disconnect fields, ensure they're preserved in the output
            if (isOneToOne === "one") {
                result[`${relation}${t}`] = yup.bool().oneOf([true], "Must be true").notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Disconnect`];
            } else {
                result[`${relation}${t}`] = yup.array().of(id).notRequired().nullable() as RelOutput<FieldName>[`${FieldName}Disconnect`];
            }
        }
        else if (t === "Update") {
            result[`${relation}${t}`] = wrap((model as YupModel<["create", "update"]>).update({ ...data, omitFields, recurseCount }), required) as RelOutput<FieldName>[`${FieldName}Update`];
        }
    }

    // For one-to-one relationships with multiple operation types, we need to ensure
    // that only one operation is provided at a time
    if (isOneToOne === "one" && relTypes.length > 1) {
        // Create tests for every pair of field names
        const fieldNames = Object.keys(result);

        // Add mutual exclusivity test to each field
        for (let i = 0; i < fieldNames.length; i++) {
            const field = fieldNames[i];
            const otherFields = fieldNames.filter(f => f !== field);

            // For each field, ensure it's exclusive with all other fields
            for (const otherField of otherFields) {
                result[field] = result[field].test({
                    name: `exclude-${field}-${otherField}`,
                    message: `Cannot provide both ${field} and ${otherField}`,
                    skipAbsent: true, // Skip this test if the field is not present
                    test(value, ctx) {
                        // If this field is present, ensure the other field is not
                        if (value !== undefined && value !== null) {
                            const parent = ctx.parent;
                            if (parent[otherField] !== undefined && parent[otherField] !== null) {
                                return false;
                            }
                        }
                        return true;
                    },
                });
            }
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
