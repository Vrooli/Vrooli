import * as yup from "yup";
import { StandardType } from "../../api/generated";
import { bool, description, enumToYup, id, jsonVariable, maxStrErr, minStrErr, name, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { resourceListValidation } from "./resourceList";
import { standardValidation } from "./standard";

const codeLanguage = yup.string().trim().removeEmptyString().min(1, minStrErr).max(128, maxStrErr);
const standardDefault = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
const standardProps = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
const standardYup = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
const variant = enumToYup(StandardType);

export const standardVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
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

export const standardVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        codeLanguage: req(codeLanguage),
        default: opt(standardDefault),
        isComplete: opt(bool),
        isFile: opt(bool),
        isPrivate: opt(bool),
        props: req(standardProps),
        variant: req(variant),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
        yup: opt(standardYup),
    }, [
        ["directoryListings", ["Connect"], "many", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["root", ["Connect", "Create"], "one", "opt", standardValidation, ["versions"]],
        ["translations", ["Create"], "many", "opt", standardVersionTranslationValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        codeLanguage: opt(codeLanguage),
        default: opt(standardDefault),
        isComplete: opt(bool),
        isFile: opt(bool),
        isPrivate: opt(bool),
        props: opt(standardProps),
        variant: opt(variant),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
        yup: opt(standardYup),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", standardValidation, ["versions"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", standardVersionTranslationValidation],
    ], [], d),
};
