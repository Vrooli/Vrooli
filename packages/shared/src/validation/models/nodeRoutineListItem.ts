import { bool, description, id, index, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { routineVersionValidation } from "./routineVersion";

export const nodeRoutineListItemTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});

export const nodeRoutineListItemValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        index: req(index),
        isOptional: opt(bool),
    }, [
        ["list", ["Connect"], "one", "req"],
        ["routineVersion", ["Connect"], "one", "req"], // Creating subroutines must be done in a separate request
        ["translations", ["Create"], "many", "opt", nodeRoutineListItemTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        isOptional: opt(bool),
    }, [
        ["routineVersion", ["Update"], "one", "opt", routineVersionValidation, ["nodes", "nodeLinks"]], // Create/update/delete of subroutines must be done in a separate request
        ["translations", ["Create", "Update", "Delete"], "many", "opt", nodeRoutineListItemTranslationValidation],
    ], [], o),
};
