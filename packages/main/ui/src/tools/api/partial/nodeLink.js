import { rel } from "../utils";
export const nodeLink = {
    __typename: "NodeLink",
    common: {
        id: true,
        operation: true,
        whens: async () => rel((await import("./nodeLinkWhen")).nodeLinkWhen, "full", { omit: "nodeLink" }),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=nodeLink.js.map