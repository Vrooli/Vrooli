import { NodeLinkWhen, NodeLinkWhenTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeLinkWhenTranslation: GqlPartial<NodeLinkWhenTranslation> = {
    __typename: "NodeLinkWhenTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const nodeLinkWhen: GqlPartial<NodeLinkWhen> = {
    __typename: "NodeLinkWhen",
    common: {
        id: true,
        condition: true,
        translations: () => rel(nodeLinkWhenTranslation, "full"),
    },
    full: {},
    list: {},
};
