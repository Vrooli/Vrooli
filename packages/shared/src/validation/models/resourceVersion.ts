/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in resourceVersion.test.ts
import * as yup from "yup";
import { ResourceSubTypeCode, ResourceSubTypeRoutine, ResourceSubTypeStandard } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, config, description, details, id, instructions, name, publicId, versionLabel, versionNotes } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { CODE_LANGUAGE_MAX_LENGTH } from "../utils/validationConstants.js";
import { resourceValidation } from "./resource.js";
import { resourceVersionRelationValidation } from "./resourceVersionRelation.js";

function resourceSubType() {
    return enumToYup({
        ...ResourceSubTypeCode,
        ...ResourceSubTypeRoutine,
        ...ResourceSubTypeStandard,
    });
}
const codeLanguage = yup.string().trim().removeEmptyString().max(CODE_LANGUAGE_MAX_LENGTH, maxStrErr);

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
        publicId: opt(publicId),
        codeLanguage: opt(codeLanguage),
        config: opt(config),
        isAutomatable: opt(bool),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        resourceSubType: opt(resourceSubType()),
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
/* c8 ignore stop */
