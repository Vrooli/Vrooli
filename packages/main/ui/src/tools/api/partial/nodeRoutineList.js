import { rel } from "../utils";
export const nodeRoutineList = {
    __typename: "NodeRoutineList",
    common: {
        id: true,
        isOrdered: true,
        isOptional: true,
        items: async () => rel((await import("./nodeRoutineListItem")).nodeRoutineListItem, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=nodeRoutineList.js.map