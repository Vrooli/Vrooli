import { GqlModelType } from "@local/shared";
import { PrismaDelegate } from "../builders/types";
import { CustomError } from "../events/error";
import { ObjectMapSingleton } from "../models/base";
import { Displayer, Duplicator, Formatter, Mutater, Searcher, Validator } from "../models/types";
import { PrismaType } from "../types";

type LogicProps = "delegate" | "display" | "duplicate" | "format" | "idField" | "mutate" | "search" | "validate";

type GetLogicReturn<
    Logic extends LogicProps,
> = {
    delegate: "delegate" extends Logic ? ((prisma: PrismaType) => PrismaDelegate) : never,
    display: "display" extends Logic ? Displayer<any> : never,
    duplicate: "duplicate" extends Logic ? Duplicator<any, any> : never,
    format: "format" extends Logic ? Formatter<any> : never,
    idField: string,
    mutate: "mutate" extends Logic ? Mutater<any> : never,
    search: "search" extends Logic ? Searcher<any, any> : never,
    validate: "validate" extends Logic ? Validator<any> : never,
}

type Undefinable<T> = {
    [K in keyof T]: T[K] | undefined;
}

/**
 * Finds the requested helper functions for working with model logic
 */
export function getLogic<
    Logic extends readonly LogicProps[],
    ThrowError extends boolean = true,
>(
    props: Logic,
    objectType: `${GqlModelType}`,
    languages: string[],
    errorTrace: string,
    throwErrorIfNotFound: ThrowError = true as ThrowError,
): ThrowError extends true ? GetLogicReturn<Logic[number]> : Undefinable<GetLogicReturn<Logic[number]>> {
    const object = ObjectMapSingleton.getInstance().map[objectType];
    // Loop through requested types to validate that all requested types exist
    for (const field of props) {
        // Get logic function
        let logic = object[field];
        // If this is for "idField" and it doesn't exist, default to "id"
        if (field === "idField" && !logic) {
            logic = "id";
            object[field] = logic;
        }
        // Make sure logic function exists
        if (!logic && throwErrorIfNotFound) {
            throw new CustomError("0367", "InvalidArgs", languages, { errorTrace, objectType, field });
        }
    }
    return object as unknown as GetLogicReturn<Logic[number]>;
}
