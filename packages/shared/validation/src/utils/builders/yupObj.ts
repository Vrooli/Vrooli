import * as yup from 'yup';
import { rel, RelationshipType } from './rel';

/**
 * Creates a yup object
 * @param fields Required and optional non-relationship fields to include in the yup object
 * @param rels 2D array of relationship data, used to create one or more fields for each relationship
 * @param excludePairs Pairs of fields which cannot both be present in the same object
 * @param omitRels Relationships to omit from the yup object
 */
export const yupObj = <T extends { [key: string]: yup.AnySchema }>(
    fields: T,
    rels: [string, readonly RelationshipType[], 'one' | 'many', 'opt' | 'req', any?, (string | string[])?][],
    excludePairs: [string, string][],
    omitRels: string | string[] | undefined,
) => {
    // Convert every relationship into yup fields
    let relObjects: { [key: string]: yup.AnySchema } = {};
    rels.forEach((params) => {
        // Skip if the relationship is in the omitRels array
        if (omitRels && (typeof omitRels === 'string' ? params[0] === omitRels : omitRels.includes(params[0]))) {
            return;
        }
        relObjects = { ...relObjects, ...rel(...params) };
    });
    // Combine fields and relObjects, 
    // and add excludePairs
    const obj = yup.object().shape({
        ...fields,
        ...relObjects,
    }, excludePairs);
    return obj;
}