import { rel } from "../utils";
export const resourceTranslation = {
    __typename: "ResourceTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};
export const resource = {
    __typename: "Resource",
    common: {
        id: true,
        index: true,
        link: true,
        usedFor: true,
        translations: () => rel(resourceTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=resource.js.map