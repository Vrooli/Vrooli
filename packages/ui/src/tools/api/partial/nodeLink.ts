import { NodeLink } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeLink: GqlPartial<NodeLink> = {
    __typename: "NodeLink",
    common: {
        id: true,
        operation: true,
        whens: async () => rel((await import("./nodeLinkWhen")).nodeLinkWhen, "full", { omit: "nodeLink" }),
    },
    full: {},
    list: {},
};
