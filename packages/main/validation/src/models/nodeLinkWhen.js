import { description, id, name, req, opt, nodeCondition, transRel, yupObj } from "../utils";
export const nodeLinkWhenTranslationValidation = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});
export const nodeLinkWhenValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        condition: req(nodeCondition),
    }, [
        ["link", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", nodeLinkWhenTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        condition: opt(nodeCondition),
    }, [
        ["link", ["Connect"], "one", "opt"],
        ["translations", ["Delete", "Create", "Update"], "many", "opt", nodeLinkWhenTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=nodeLinkWhen.js.map