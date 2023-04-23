import { bool, description, id, index, name, opt, req, transRel, yupObj } from "../utils";
import { routineVersionValidation } from "./routineVersion";
export const nodeRoutineListItemTranslationValidation = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});
export const nodeRoutineListItemValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        index: req(index),
        isOptional: opt(bool),
    }, [
        ["list", ["Connect"], "one", "req"],
        ["routineVersion", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", nodeRoutineListItemTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        isOptional: opt(bool),
    }, [
        ["routineVersion", ["Update"], "one", "opt", routineVersionValidation, ["nodes", "nodeLinks"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", nodeRoutineListItemTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=nodeRoutineListItem.js.map