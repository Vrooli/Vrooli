import * as yup from "yup";
import { bool, description, id, maxStrErr, name, opt, req, transRel, YupModel, yupObj } from "../utils";

export const childOrder = yup.string().trim().removeEmptyString().max(4096, maxStrErr);

export const projectVersionDirectoryTranslationValidation: YupModel = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const projectVersionDirectoryValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ["parentDirectory", ["Connect"], "one", "opt"],
        ["projectVersion", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", projectVersionDirectoryTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ["parentDirectory", ["Connect", "Disconnect"], "one", "opt"],
        ["projectVersion", ["Connect"], "one", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", projectVersionDirectoryTranslationValidation],
    ], [], d),
};
