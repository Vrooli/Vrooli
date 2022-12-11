import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { Formatter, GraphQLModelType } from "../models/types";

export function getFormatter<
    GQLObject extends { [x: string]: any },
    SupplementalFields extends readonly (keyof GQLObject extends infer R ? R extends string ? R : never : never)[]
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Formatter<GQLObject, SupplementalFields> {
    const formatter: Formatter<GQLObject, SupplementalFields> | undefined = ObjectMap[objectType]?.format;
    if (!formatter) {
        throw new CustomError('0339', 'InternalError', languages, { errorTrace, objectType });
    }
    return formatter;
}