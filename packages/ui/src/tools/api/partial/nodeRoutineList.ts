import { NodeRoutineList } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeRoutineList: GqlPartial<NodeRoutineList> = {
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
