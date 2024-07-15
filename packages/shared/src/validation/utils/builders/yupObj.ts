import * as yup from "yup";
import { splitDotNotation } from "../../../utils";
import { omit } from "../../../utils/omit";
import { YupModel, YupModelOptions, YupMutateParams } from "../types";
import { RelationshipType, rel } from "./rel";

/**
 * Creates a yup object
 * @param primFields Required and optional primitive (non-relationship) fields to include in the yup object
 * @param rels 2D array of relationship data, used to create one or more fields for each relationship
 * @param requireOneGroups Groups of fields which exactly one field must be present
 * @param data Parameters for YupModel create and update
 */
export function yupObj<AllFields extends { [key: string]: yup.AnySchema }>(
    primFields: Partial<AllFields>,
    rels: [
        string, // Relationship name
        readonly RelationshipType[], // Operations allowed on this relationship (e.g. "Create", "Delete")
        "one" | "many", // If this relationship is one-to-one or one-to-many, which is used to determine if we should use an array or object
        "opt" | "req", // If this relationship is optional or required, which is used to determine if we should use an opt() or req()
        YupModel<YupModelOptions[]>?, // The YupModel (validation info) for this relationship
        string[]? // Any fields which should be omitted from the relationsip's validation object. Typically used to prevent circular references
    ][],
    requireOneGroups: [string, string, boolean][], // NOTE: All of these fields must be marked as optional, since we can't override individual field required validations
    data: YupMutateParams,
) {
    // Find fields which should be omitted from the top level object
    const [topFields] = splitDotNotation(data.omitFields ?? []);
    // Convert every relationship into yup fields
    let relFields: Partial<AllFields> = {};
    rels.forEach((params) => {
        // If relationship is in topFields, skip it
        if (topFields.includes(params[0])) return;
        // Filter out any relationship types where params[0] or `${params[0]}${type}` is in topFields
        const filteredRelTypes = params[1].filter((type) => !topFields.includes(`${params[0]}${type}`));
        // If there are no relationship types left, skip this relationship
        if (!filteredRelTypes.length) return;
        // Find nested omitFields for this relationship. Can either start with `${params[0]}.` or `${params[0]}${type}.`
        const nestedFieldStarts = [`${params[0]}.`, ...filteredRelTypes.map((type) => `${params[0]}${type}.`)];
        const fullNestedFields = Object.keys(data.omitFields ?? []).filter((field) => nestedFieldStarts.some((start) => field.startsWith(start)));
        const [, relNestedFields] = splitDotNotation(fullNestedFields);
        // Create relationship
        const relResult = rel({
            ...data,
            omitFields: [...relNestedFields, ...(params.length === 6 && Array.isArray(params[5]) ? params[5] : [])],
            recurseCount: (data.recurseCount ?? 0) + 1,
        }, ...params);
        // Add to relFields
        relFields = { ...relFields, ...relResult };
    });
    // Combine fields and omit unwanted ones
    const filteredFields = omit({
        ...primFields,
        ...relFields,
    }, topFields) as AllFields;
    // Create yup object
    let schema = yup.object().shape(filteredFields);
    // Create tests for each requireOneGroup
    requireOneGroups.forEach(([field1, field2, isOneRequired]) => {
        schema = schema.test(
            `exclude-${field1}-${field2}`,
            `Only one of the following fields can be present: ${field1}, ${field2}`,
            function requireTest(value) {
                // If the value (meaning the value of the full object, not the fields being tested) is null or undefined, 
                // we shouldn't be here. So we'll return true to pass the test
                if (value === null || value === undefined) return true;
                // Count the number of fields which are present
                const fieldCounts = typeof value === "object" ? [field1, field2].filter((field) => value[field] !== undefined).length : 0;
                // While we're here, we'll check if any of the fields were marked as required.  
                // If so, it will always fail. So we'll give a warning
                const anyRequired = [field1, field2].some((field) => this.schema.fields[field].tests.some((test) => test.OPTIONS.name === "required"));
                if (anyRequired) {
                    console.warn(`One of the following fields is marked as required, so this require-one test will always fail: ${field1}, ${field2}`);
                }
                return isOneRequired ? fieldCounts === 1 : fieldCounts <= 1;
            },
        );
    });
    return schema;
}
