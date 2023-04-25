import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { nodeRoutineListItemValidation } from "./nodeRoutineListItem";

export const nodeRoutineListValidation: YupModel = {
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
