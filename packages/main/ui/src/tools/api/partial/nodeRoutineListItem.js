import { rel } from "../utils";
export const nodeRoutineListItemTranslation = {
    __typename: "NodeRoutineListItemTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};
export const nodeRoutineListItem = {
    __typename: "NodeRoutineListItem",
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => rel(nodeRoutineListItemTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=nodeRoutineListItem.js.map