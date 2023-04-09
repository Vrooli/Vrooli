import * as yup from 'yup';
import { id } from '../commonFields';
import { YupModel } from '../types';
import { opt } from './opt';
import { optArr } from './optArr';
import { req } from './req';
import { reqArr } from './reqArr';

export type RelationshipType = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Update';


// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends 'object' | 'id', IsOneToOne extends 'one' | 'many'> =
    T extends 'object' ?
    IsOneToOne extends 'one' ? yup.ObjectSchema<any> : yup.ArraySchema<any> :
    IsOneToOne extends 'one' ? yup.StringSchema : yup.ArraySchema<any>;

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<IsOneToOne extends 'one' | 'many'> =
    IsOneToOne extends 'one' ? yup.BooleanSchema : yup.ArraySchema<any>;

type RelOutput<
    IsOneToOne extends 'one' | 'many',
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: 'Connect' extends RelTypes ? MaybeArray<'id', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: 'Create' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Delete`]: 'Delete' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Disconnect`]: 'Disconnect' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Update`]: 'Update' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never })
    )

/**
 * Creates the validation fields for a relationship
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param isRequired "opt" or "req" to mark the fields as optional or required. 
 * If required, only one "Connect" or "Create" is actually marked as required, and the other 
 * fields are marked as optional. If optional, all fields are marked as optional.
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @param model The relationship's validation object
 * @parm omitFields An array of fields to omit from the validation object
 * @returns An object with the validation fields for the relationship
 */
export const rel = <
    IsOneToOne extends 'one' | 'many',
    IsRequired extends 'opt' | 'req',
    RelTypes extends readonly RelationshipType[],
    FieldName extends string,
    // Model only required when RelTypes includes 'Create' or 'Update'
    Model extends ('Create' extends RelTypes[number] ?
        'Update' extends RelTypes[number] ?
        YupModel<true, true> :
        YupModel<true, false> :
        'Update' extends RelTypes[number] ?
        YupModel<false, true> :
        never),
    OmitField extends string,
>(
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    isRequired: IsRequired,
    // model only required if relTypes includes 'Create' or 'Update'
    model?: Model,
    omitFields?: OmitField | OmitField[],
): RelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Check if model is required
    if (relTypes.includes('Create') || relTypes.includes('Update')) {
        if (!model) throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through relation types
    for (const t of relTypes) {
        // Determine if field is required. If both 'Connect' and 'Create' are allowed, both 
        // are marked as optional here. This is because yup defines this one-of rule in the second
        // parameter of object.shape()
        // Count the number of times 'Connect' and 'Create' are in relTypes
        const connectCreateCount = relTypes.filter(x => x === 'Connect' || x === 'Create').length;
        const required = isRequired === 'req' && connectCreateCount === 1 && (t === 'Connect' || t === 'Create');
        // Add validation field to result
        if (t === 'Connect') {
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                required ? req(id) : opt(id) :
                required ? reqArr(id) : optArr(id);
        }
        else if (t === 'Create') {
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                required ? req((model as YupModel<true, true>).create({ o: omitFields })) : opt((model as YupModel<true, true>).create({ o: omitFields })) :
                required ? reqArr((model as YupModel<true, true>).create({ o: omitFields })) : optArr((model as YupModel<true, true>).create({ o: omitFields }));
        }
        else if (t === 'Delete') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt(yup.bool()) : optArr(id);
        }
        else if (t === 'Disconnect') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt(yup.bool()) : optArr(id);
        }
        else if (t === 'Update') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt((model as YupModel<true, true>).update({ o: omitFields })) : optArr((model as YupModel<true, true>).update({ o: omitFields }));
        }
    }
    // Return result
    return result as any;
};