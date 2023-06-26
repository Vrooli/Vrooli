import { GqlModelType } from "@local/shared";
import { CustomError } from "../events";
import { ObjectMap } from "../models/base";
import { ModelLogic } from "../models/types";

type LogicProps = "delegate" | "display" | "duplicate" | "format" | "idField" | "mutate" | "search" | "validate";

type GetLogicReturn<
    Logic extends LogicProps,
> = {
    delegate: "delegate" extends Logic ? Required<ModelLogic<any, any>>["delegate"] : ModelLogic<any, any>["delegate"],
    display: "display" extends Logic ? Required<ModelLogic<any, any>>["display"] : ModelLogic<any, any>["display"],
    duplicate: "duplicate" extends Logic ? Required<ModelLogic<any, any>>["duplicate"] : ModelLogic<any, any>["duplicate"],
    format: "format" extends Logic ? Required<ModelLogic<any, any>>["format"] : ModelLogic<any, any>["format"],
    idField: string,
    mutate: "mutate" extends Logic ? Required<ModelLogic<any, any>>["mutate"] : ModelLogic<any, any>["mutate"],
    search: "search" extends Logic ? Required<ModelLogic<any, any>>["search"] : ModelLogic<any, any>["search"],
    validate: "validate" extends Logic ? Required<ModelLogic<any, any>>["validate"] : ModelLogic<any, any>["validate"],
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
    // Make sure object exists in map
    const object = ObjectMap[objectType];
    if (!object) {
        if (throwErrorIfNotFound) throw new CustomError("0280", "InvalidArgs", languages, { errorTrace, objectType });
        const result: any = props.reduce((acc, cur) => {
            acc[cur] = undefined;
            return acc;
        }, {} as Undefinable<GetLogicReturn<Logic[number]>>);
        return result as GetLogicReturn<Logic[number]>;
    }
    // If no props are requested, return the entire object
    if (!props.length) return object as GetLogicReturn<Logic[number]>;
    // Loop through requested types to validate that all requested types exist
    for (const field of props) {
        // Get logic function
        let logic = object[field as any];
        // If this is for "idField" and it doesn't exist, default to "id"
        if (field === "idField" && !logic) {
            logic = "id";
            object[field as any] = logic;
        }
        // Make sure logic function exists
        if (!logic && throwErrorIfNotFound) {
            throw new CustomError("0367", "InvalidArgs", languages, { errorTrace, objectType, field });
        }
    }
    return object as GetLogicReturn<Logic[number]>;
}
