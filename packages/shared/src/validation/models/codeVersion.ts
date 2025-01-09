import * as yup from "yup";
import { CodeType } from "../../api/types";
import { YupModel, bool, description, enumToYup, id, jsonVariable, maxStrErr, minStrErr, name, opt, req, transRel, versionLabel, versionNotes, yupObj } from "../utils";
import { codeValidation } from "./code";
import { resourceListValidation } from "./resourceList";

const codeDefault = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
const codeLanguage = yup.string().trim().removeEmptyString().min(1, minStrErr).max(128, maxStrErr);
const codeType = enumToYup(CodeType);
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
        codeLanguage: req(codeLanguage),
        codeType: req(codeType),
        content: req(content),
        default: opt(codeDefault),
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
        codeLanguage: opt(codeLanguage),
        content: opt(content),
        default: opt(codeDefault),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", codeValidation, ["versions"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", codeVersionTranslationValidation],
    ], [], d),
};
