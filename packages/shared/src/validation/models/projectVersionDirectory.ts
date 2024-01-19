import * as yup from "yup";
import { bool, description, id, maxStrErr, name, opt, req, transRel, YupModel, yupObj } from "../utils";

export const childOrder = yup.string().trim().removeEmptyString().max(4096, maxStrErr);

export const projectVersionDirectoryTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const projectVersionDirectoryValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ["parentDirectory", ["Connect"], "one", "opt"],
        ["projectVersion", ["Connect"], "one", "req"],
        ["childApiVersions", ["Connect"], "many", "opt"],
        ["childNoteVersions", ["Connect"], "many", "opt"],
        ["childOrganizations", ["Connect"], "many", "opt"],
        ["childProjectVersions", ["Connect"], "many", "opt"],
        ["childRoutineVersions", ["Connect"], "many", "opt"],
        ["childSmartContractVersions", ["Connect"], "many", "opt"],
        ["childStandardVersions", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", projectVersionDirectoryTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ["parentDirectory", ["Connect", "Disconnect"], "one", "opt"],
        ["projectVersion", ["Connect"], "one", "opt"],
        ["childApiVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["childNoteVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["childOrganizations", ["Connect", "Disconnect"], "many", "opt"],
        ["childProjectVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["childRoutineVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["childSmartContractVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["childStandardVersions", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", projectVersionDirectoryTranslationValidation],
    ], [], d),
};
