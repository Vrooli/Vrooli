import { description, id, name, opt, permissions, req, transRel, YupModel, yupObj } from "../utils";

export const roleTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: req(description),
    }),
    update: () => ({
        description: opt(description),
    }),
});

export const roleValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect"], "many", "opt"],
        ["organization", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", roleTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        name: req(name),
        permissions: opt(permissions),
    }, [
        ["members", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", roleTranslationValidation],
    ], [], d),
};
