import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, name, versionLabel, versionNotes } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { projectValidation } from "./project.js";
import { projectVersionDirectoryValidation } from "./projectVersionDirectory.js";

export const projectVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const projectVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        isComplete: opt(bool),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "opt", projectValidation, ["versions"]],
        ["translations", ["Create"], "many", "opt", projectVersionTranslationValidation],
        ["directories", ["Create"], "many", "opt", projectVersionDirectoryValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        isComplete: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Update"], "one", "opt", projectValidation, ["versions"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", projectVersionTranslationValidation],
        ["directories", ["Create", "Update", "Delete"], "many", "opt", projectVersionDirectoryValidation],
    ], [], d),
};
