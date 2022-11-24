import { CustomError } from "../../events";
import { ObjectMap } from "../builder";
import { Formatter, GraphQLModelType } from "../types";

export function getFormatter<
    GQLObject extends { [x: string]: any },
    SupplementalFields extends string
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