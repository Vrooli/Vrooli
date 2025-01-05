import { NodeLink } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeLink: GqlPartial<NodeLink> = {
    common: {
        id: true,
        from: {
            id: true,
        },
        operation: true,
        to: {
            id: true,
        },
        whens: async () => rel((await import("./nodeLinkWhen")).nodeLinkWhen, "full", { omit: "nodeLink" }),
    },
};
