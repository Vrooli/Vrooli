import { NodeLinkWhen, NodeLinkWhenTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const nodeLinkWhenTranslation: ApiPartial<NodeLinkWhenTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const nodeLinkWhen: ApiPartial<NodeLinkWhen> = {
    common: {
        id: true,
        condition: true,
        translations: () => rel(nodeLinkWhenTranslation, "full"),
    },
};
