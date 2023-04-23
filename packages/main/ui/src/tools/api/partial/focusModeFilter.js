import { rel } from "../utils";
export const focusModeFilter = {
    __typename: "FocusModeFilter",
    full: {
        id: true,
        filterType: true,
        tag: async () => rel((await import("./tag")).tag, "list"),
        focusMode: async () => rel((await import("./focusMode")).focusMode, "list", { omit: "filters" }),
    },
};
//# sourceMappingURL=focusModeFilter.js.map