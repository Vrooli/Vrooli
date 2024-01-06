import * as yup from "yup";
import { YupModel, YupModelOptions, YupMutateParams } from "../types";
import { RelationshipType, rel } from "./rel";

/**
 * Creates a yup object
 * @param fields Required and optional non-relationship fields to include in the yup object
 * @param rels 2D array of relationship data, used to create one or more fields for each relationship
 * @param excludePairs Pairs of fields which cannot both be present in the same object
 * @param data Parameters for YupModel create and update
 */
export const yupObj = <T extends { [key: string]: yup.AnySchema }>(
    fields: T,
    rels: [string, readonly RelationshipType[], "one" | "many", "opt" | "req", YupModel<YupModelOptions[]>?, (string | string[])?][],
    excludePairs: [string, string][],
    data: YupMutateParams,
) => {
    // Convert every relationship into yup fields
    let relObjects: { [key: string]: yup.AnySchema } = {};
    rels.forEach((params) => {
        // Skip if the relationship is in the omitRels array
        if (data.omitRels && (typeof data.omitRels === "string" ? params[0] === data.omitRels : data.omitRels.includes(params[0]))) {
            return;
        }
        relObjects = { ...relObjects, ...rel(data, ...params) };
    });
    // Combine fields and relObjects, 
    // and add excludePairs
    const obj = yup.object().shape({
        ...fields,
        ...relObjects,
    }, excludePairs);
    return obj;
};
