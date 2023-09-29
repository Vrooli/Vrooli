import { description, id, name, nodeCondition, opt, req, transRel, YupModel, yupObj } from "../utils";

export const nodeLinkWhenTranslationValidation: YupModel = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const nodeLinkWhenValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        condition: req(nodeCondition),
    }, [
        ["link", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", nodeLinkWhenTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        condition: opt(nodeCondition),
    }, [
        ["link", ["Connect"], "one", "opt"],
        ["translations", ["Delete", "Create", "Update"], "many", "opt", nodeLinkWhenTranslationValidation],
    ], [], d),
};
