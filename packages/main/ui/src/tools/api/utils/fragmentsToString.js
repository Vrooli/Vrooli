import { partialToStringHelper } from "./partialToStringHelper";
export const fragmentsToString = async (fragments) => {
    console.log("fragmentsToString start", Object.keys(fragments), "\n\n");
    const result = [];
    for (const [name, partial] of Object.entries(fragments)) {
        const objectType = name.split("_")[0];
        let fragmentString = "";
        fragmentString += `fragment ${name} on ${objectType} {\n`;
        fragmentString += await partialToStringHelper(partial);
        fragmentString += "}";
        result.push([name, fragmentString]);
    }
    console.log("fragmentsToString result", result.map(([name, tag]) => name), "\n\n");
    return result;
};
//# sourceMappingURL=fragmentsToString.js.map