import * as yup from "yup";
import { bool, description, id, jsonVariable, maxStrErr, name, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { codeValidation } from "./code";
import { resourceListValidation } from "./resourceList";

const codeDefault = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
const contractType = yup.string().trim().removeEmptyString().max(256, maxStrErr);
const content = yup.string().trim().removeEmptyString().max(8192, maxStrErr);

export const codeVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        jsonVariable: opt(jsonVariable),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        jsonVariable: opt(jsonVariable),
        name: opt(name),
    }),
});

export const codeVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isPrivate: opt(bool),
        default: opt(codeDefault),
        contractType: req(contractType),
        content: req(content),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect"], "many", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["root", ["Connect", "Create"], "one", "opt", codeValidation, ["versions"]],
        ["translations", ["Create"], "many", "opt", codeVersionTranslationValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isPrivate: opt(bool),
        default: opt(codeDefault),
        contractType: opt(contractType),
        content: opt(content),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", codeValidation, ["versions"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", codeVersionTranslationValidation],
    ], [], d),
};
