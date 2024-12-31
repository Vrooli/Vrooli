import { FocusModeFilter } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const focusModeFilter: GqlPartial<FocusModeFilter> = {
    __typename: "FocusModeFilter",
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import("./tag")).tag, "list"),
    },
};
