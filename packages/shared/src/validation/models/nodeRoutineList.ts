import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { nodeRoutineListItemValidation } from "./nodeRoutineListItem";

export const nodeRoutineListValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        isOrdered: opt(bool),
        isOptional: opt(bool),
    }, [
        ["items", ["Create"], "many", "opt", nodeRoutineListItemValidation],
        ["node", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        isOrdered: opt(bool),
        isOptional: opt(bool),
    }, [
        ["items", ["Create", "Update", "Delete"], "many", "opt", nodeRoutineListItemValidation],
    ], [], d),
};
