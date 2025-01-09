import { NodeRoutineList } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const nodeRoutineList: ApiPartial<NodeRoutineList> = {
    common: {
        id: true,
        isOrdered: true,
        isOptional: true,
        items: async () => rel((await import("./nodeRoutineListItem")).nodeRoutineListItem, "full"),
    },
};
