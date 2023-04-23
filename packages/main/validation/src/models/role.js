import { description, id, name, opt, permissions, req, transRel, yupObj } from "../utils";
export const roleTranslationValidation = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
    },
});
export const roleValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect"], "many", "opt"],
        ["organization", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", roleTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", roleTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=role.js.map