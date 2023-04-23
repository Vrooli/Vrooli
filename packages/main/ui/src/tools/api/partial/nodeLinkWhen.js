import { rel } from "../utils";
export const nodeLinkWhenTranslation = {
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
export const nodeLinkWhen = {
    __typename: "NodeLinkWhen",
    common: {
        id: true,
        condition: true,
        translations: () => rel(nodeLinkWhenTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=nodeLinkWhen.js.map