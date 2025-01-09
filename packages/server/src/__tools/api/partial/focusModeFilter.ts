import { FocusModeFilter } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const focusModeFilter: ApiPartial<FocusModeFilter> = {
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import("./tag")).tag, "list"),
    },
};
