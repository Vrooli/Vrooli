import { NodeLinkWhen, NodeLinkWhenTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeLinkWhenTranslation: GqlPartial<NodeLinkWhenTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const nodeLinkWhen: GqlPartial<NodeLinkWhen> = {
    common: {
        id: true,
        condition: true,
        translations: () => rel(nodeLinkWhenTranslation, "full"),
    },
};
