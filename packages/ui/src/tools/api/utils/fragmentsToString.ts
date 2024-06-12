import { DeepPartialBooleanWithFragments } from "../types";
import { partialToStringHelper } from "./partialToStringHelper";

/**
 * Converts fragment data (from DeepPartialBooleanWithFragments.__define) into graphql-tag strings.
 * @param fragments The fragment data.
 * @returns a graphql-tag string for each fragment, along with its name. (Array<[name, tag]>)
 */
export const fragmentsToString = async (
    fragments: Exclude<DeepPartialBooleanWithFragments<any>["__define"], undefined>,
) => {
    // Initialize result
    const result: [string, string][] = [];
    // Loop through fragments
    for (const [name, partial] of Object.entries(fragments)) {
        const objectType = name.split("_")[0];
        let fragmentString = "";
        fragmentString += `fragment ${name} on ${objectType} {\n`;
        fragmentString += await partialToStringHelper(partial as any);
        fragmentString += "}";
        result.push([name, fragmentString]);
    }
    return result;
};
