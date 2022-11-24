import { CustomError } from "../../events";
import { ObjectMap } from "../builder";
import { Displayer, GraphQLModelType } from "../types";

export function getDisplay(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Displayer {
    const display: Displayer | undefined = ObjectMap[objectType]?.display
    if (!display) {
        throw new CustomError('0344', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    return display;
}