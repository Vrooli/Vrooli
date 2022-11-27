import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { Displayer, GraphQLModelType } from "../models/types";

export function getDisplayer<PrismaSelect extends { [x: string]: any; }, PrismaSelectData extends { [x: string]: any; }>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Displayer<PrismaSelect, PrismaSelectData> {
    const display: Displayer<PrismaSelect, PrismaSelectData> | undefined = ObjectMap[objectType]?.display
    if (!display) {
        throw new CustomError('0344', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    return display;
}