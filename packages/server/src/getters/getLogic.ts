import { GqlModelType } from "@local/shared";
import { CustomError } from "../events";
import { ObjectMap } from "../models/base";
import { ModelLogic } from "../models/types";

type LogicProps = "delegate" | "display" | "duplicate" | "format" | "mutate" | "search" | "validate";

type GetLogicReturn<
    Logic extends LogicProps,
> = {
    delegate: "delegate" extends Logic ? Required<ModelLogic<any, any>>["delegate"] : ModelLogic<any, any>["delegate"],
    display: "display" extends Logic ? Required<ModelLogic<any, any>>["display"] : ModelLogic<any, any>["display"],
    duplicate: "duplicate" extends Logic ? Required<ModelLogic<any, any>>["duplicate"] : ModelLogic<any, any>["duplicate"],
    format: "format" extends Logic ? Required<ModelLogic<any, any>>["format"] : ModelLogic<any, any>["format"],
    mutate: "mutate" extends Logic ? Required<ModelLogic<any, any>>["mutate"] : ModelLogic<any, any>["mutate"],
    search: "search" extends Logic ? Required<ModelLogic<any, any>>["search"] : ModelLogic<any, any>["search"],
    validate: "validate" extends Logic ? Required<ModelLogic<any, any>>["validate"] : ModelLogic<any, any>["validate"],
}

/**
 * Finds the requested helper functions for working with model logic
 */
export function getLogic<
    Logic extends readonly LogicProps[],
>(
    props: Logic,
    objectType: `${GqlModelType}`,
    languages: string[],
    errorTrace: string,
): GetLogicReturn<Logic[number]> {
    // Make sure object exists in map
    const object = ObjectMap[objectType];
    if (!object) throw new CustomError("0280", "InvalidArgs", languages, { errorTrace, objectType });
    // If no props are requested, return the entire object
    if (!props.length) return object as GetLogicReturn<Logic[number]>;
    // Loop through requested types to validate that all requested types exist
    for (const field of props) {
        // Get logic function
        const logic = object[field as any];
        // Make sure logic function exists
        if (!logic) throw new CustomError("0367", "InvalidArgs", languages, { errorTrace, objectType, field });
    }
    return object as GetLogicReturn<Logic[number]>;
}
