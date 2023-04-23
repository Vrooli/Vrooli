import { exists } from "@local/utils";
export const noNull = (...args) => {
    for (const arg of args) {
        if (exists(arg)) {
            return arg;
        }
    }
    return undefined;
};
//# sourceMappingURL=noNull.js.map