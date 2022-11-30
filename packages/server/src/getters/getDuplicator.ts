import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { Duplicator, GraphQLModelType } from "../models/types";

export function getDuplicator<
    Select extends { id?: boolean | undefined, intendToPullRequest?: boolean | undefined, [x: string]: any },
    Data extends { [x: string]: any }
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Duplicator<Select, Data> {
    throw new CustomError('0348', 'NotImplemented', languages, { errorTrace, objectType });
    // const duplicate: Duplicator<Select, Data> | undefined = ObjectMap[objectType]?.duplicate
    // if (!duplicate) {
    //     throw new CustomError('0348', 'InvalidArgs', languages, { errorTrace, objectType });
    // }
    // return duplicate;
}