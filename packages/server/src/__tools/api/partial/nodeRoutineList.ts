import { NodeRoutineList } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeRoutineList: GqlPartial<NodeRoutineList> = {
    common: {
        id: true,
        isOrdered: true,
        isOptional: true,
        items: async () => rel((await import("./nodeRoutineListItem")).nodeRoutineListItem, "full"),
    },
};
