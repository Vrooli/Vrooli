import { bool, id, req, opt, yupObj } from "../utils";
import { nodeRoutineListItemValidation } from "./nodeRoutineListItem";
export const nodeRoutineListValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        isOrdered: opt(bool),
        isOptional: opt(bool),
    }, [
        ["items", ["Create"], "many", "opt", nodeRoutineListItemValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isOrdered: opt(bool),
        isOptional: opt(bool),
    }, [
        ["items", ["Create", "Update", "Delete"], "many", "opt", nodeRoutineListItemValidation],
    ], [], o),
};
//# sourceMappingURL=nodeRoutineList.js.map