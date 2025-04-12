import { FocusModeFilter } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const focusModeFilter: ApiPartial<FocusModeFilter> = {
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import("./tag.js")).tag, "list"),
    },
};
