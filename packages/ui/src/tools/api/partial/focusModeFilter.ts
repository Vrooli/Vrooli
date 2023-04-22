import { FocusModeFilter } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const focusModeFilter: GqlPartial<FocusModeFilter> = {
    __typename: "FocusModeFilter",
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import("./tag")).tag, "list"),
        focusMode: async () => rel((await import("./focusMode")).focusMode, "list", { omit: "filters" }),
    },
}