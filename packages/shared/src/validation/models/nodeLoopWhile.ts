import { description, id, name, nodeCondition, opt, req, transRel, YupModel, yupObj } from "../utils";

export const nodeLoopWhileTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const nodeLoopWhileValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        condition: req(nodeCondition),
    }, [
        ["to", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", nodeLoopWhileTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        condition: opt(nodeCondition),
    }, [
        ["to", ["Connect"], "one", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", nodeLoopWhileTranslationValidation],
    ], [], d),
};
