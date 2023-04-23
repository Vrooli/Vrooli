import { exists } from "@local/utils";
import { fragmentsNeeded } from "./fragmentsNeeded";
import { fragmentsToString } from "./fragmentsToString";
import { partialToStringHelper } from "./partialToStringHelper";
import { rel } from "./rel";
export const partialToString = async ({ endpointType, endpointName, indent, inputType, partial, selectionType, }) => {
    let fragments = [];
    let tag = "";
    let combined = {};
    if (exists(partial) && exists(selectionType)) {
        combined = await rel(partial, selectionType);
    }
    const { __define, ...rest } = combined;
    tag += `
${" ".repeat(indent)}${endpointType} ${endpointName}`;
    if (exists(inputType)) {
        tag += `($input: ${inputType}!)`;
    }
    tag += ` {
${" ".repeat(indent + 2)}`;
    tag += `${endpointName}${exists(inputType) ? "(input: $input)" : ""}`;
    if (exists(rest) && Object.keys(rest).length > 0) {
        tag += ` {
`;
        tag += await partialToStringHelper(rest, indent + 4);
        tag += `${" ".repeat(indent + 2)}}`;
    }
    tag += `
${" ".repeat(indent)}}`;
    if (exists(__define) && Object.keys(__define).length > 0) {
        let fragmentsString = "";
        fragments = await fragmentsToString(__define);
        fragments = fragmentsNeeded(fragments, tag);
        fragments.sort(([a], [b]) => a.localeCompare(b));
        fragments.forEach(([fragmentName]) => {
            fragmentsString += `${" ".repeat(indent)}\${${fragmentName}}\n`;
        });
        tag = fragmentsString + tag;
    }
    return { fragments, tag };
};
//# sourceMappingURL=partialToString.js.map