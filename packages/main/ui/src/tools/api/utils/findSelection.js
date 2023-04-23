import { exists } from "@local/utils";
export const findSelection = (obj, selection) => {
    let result;
    if (selection === "common")
        result = exists(obj.common) ? "common" : exists(obj.list) ? "list" : exists(obj.full) ? "full" : exists(obj.nav) ? "nav" : undefined;
    else if (selection === "list")
        result = exists(obj.list) ? "list" : exists(obj["common"]) ? "common" : exists(obj.full) ? "full" : exists(obj.nav) ? "nav" : undefined;
    else if (selection === "full")
        result = exists(obj.full) ? "full" : exists(obj.list) ? "list" : exists(obj["common"]) ? "common" : exists(obj.nav) ? "nav" : undefined;
    else if (selection === "nav")
        result = exists(obj.nav) ? "nav" : exists(obj["common"]) ? "common" : exists(obj.list) ? "list" : exists(obj.full) ? "full" : undefined;
    if (!exists(result)) {
        throw new Error(`Could not determine actual selection type for '${obj.__typename}' '${selection}'`);
    }
    if (result !== selection) {
        console.warn(`Specified selection type '${selection}' for '${obj.__typename}' does not exist. Using '${result}' instead.`);
    }
    return result;
};
//# sourceMappingURL=findSelection.js.map