import * as yup from "yup";
import { splitDotNotation } from "../../../utils/objects.js";
import { omit } from "../../../utils/omit.js";
import { type YupModel, type YupModelOptions, type YupMutateParams } from "../types.js";
import { type RelationshipType, rel } from "./rel.js";

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
    // Only include fields that don't contain dots (i.e., true top-level fields)
    const topLevelOmitFields = (data.omitFields ?? []).filter(field => !field.includes("."));
    const [topFields] = splitDotNotation(topLevelOmitFields);
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
        const fullNestedFields = (data.omitFields ?? []).filter((field) => nestedFieldStarts.some((start) => field.startsWith(start)));
        const [, relNestedFields] = splitDotNotation(fullNestedFields);
        // Create relationship
        const relResult = rel({
            ...data,
            // eslint-disable-next-line no-magic-numbers
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
        // Skip constraint if either field is omitted from the schema
        const field1InSchema = filteredFields[field1] !== undefined;
        const field2InSchema = filteredFields[field2] !== undefined;
        if (!field1InSchema || !field2InSchema) {
            return; // Skip this constraint entirely when either field is omitted
        }
        
        schema = schema.test(
            `exclude-${field1}-${field2}`,
            `Only one of the following fields can be present: ${field1}, ${field2}`,
            function requireTest(value) {
                // If the value (meaning the value of the full object, not the fields being tested) is null or undefined, 
                // we shouldn't be here. So we'll return true to pass the test
                if (value === null || value === undefined) return true;

                // Count the number of fields which are present
                // Field is present if it's defined and not null
                const fieldCounts = typeof value === "object" ?
                    [field1, field2].filter((field) => value[field] !== undefined && value[field] !== null).length : 0;

                // While we're here, we'll check if any of the fields were marked as required.  
                // If so, it will always fail. So we'll give a warning
                const anyRequired = [field1, field2].some((field) =>
                    this.schema?.fields?.[field]?.tests?.some((test) => test.OPTIONS?.name === "required"));

                if (anyRequired) {
                    console.warn(`[yupObj] One of the following fields is marked as required, so this require-one test will always fail: ${field1}, ${field2}`);
                }

                // Check if fields actually exist in the value object
                // If no fields are present and one is required, it might mean the relationship 
                // is being handled automatically by a parent, so we should pass the test
                if (isOneRequired) {
                    // If both fields are missing from the schema, the relationship is being handled automatically
                    const bothFieldsMissingFromSchema = ![field1, field2].some((field) => 
                        this.schema?.fields?.[field] !== undefined);
                    if (bothFieldsMissingFromSchema) {
                        return true; // Pass the test when relationship is handled automatically
                    }
                    return fieldCounts === 1;
                } else {
                    return fieldCounts <= 1;
                }
            },
        );
    });
    return schema;
}
