import * as yup from "yup";
import { ResourceSubTypeCode, ResourceSubTypeRoutine, ResourceSubTypeStandard } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, config, description, details, id, instructions, name, versionLabel, versionNotes } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { resourceValidation } from "./resource.js";
import { resourceVersionRelationValidation } from "./resourceVersionRelation.js";

const resourceSubType = enumToYup({
    ...ResourceSubTypeCode,
    ...ResourceSubTypeRoutine,
    ...ResourceSubTypeStandard,
});
const codeLanguage = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const resourceVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        details: opt(details),
        instructions: opt(instructions),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        details: opt(details),
        instructions: opt(instructions),
        name: opt(name),
    }),
});

export const resourceVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        codeLanguage: opt(codeLanguage),
        config: opt(config),
        isAutomatable: opt(bool),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        resourceSubType: req(resourceSubType),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "opt", resourceValidation, ["versions"]],
        ["relatedVersions", ["Create"], "many", "opt", resourceVersionRelationValidation, ["fromVersion"]],
        ["translations", ["Create"], "many", "opt", resourceVersionTranslationValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        codeLanguage: opt(codeLanguage),
        config: opt(config),
        isAutomatable: opt(bool),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Update"], "one", "opt", resourceValidation, ["versions"]],
        ["relatedVersions", ["Create", "Update", "Delete"], "many", "opt", resourceVersionRelationValidation, ["fromVersion"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceVersionTranslationValidation],
    ], [], d),
};
