import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, details, id, name, summary, url, versionLabel, versionNotes } from "../utils/commonFields.js";
import { maxStrErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { apiValidation } from "./api.js";
import { resourceListValidation } from "./resourceList.js";

// eslint-disable-next-line no-magic-numbers
export const schemaLanguage = yup.string().trim().removeEmptyString().min(1, minStrErr).max(128, maxStrErr);
// eslint-disable-next-line no-magic-numbers
export const schemaText = yup.string().trim().removeEmptyString().min(3, minStrErr).max(16384, maxStrErr);

export const apiVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        name: req(name),
        details: opt(details),
        summary: opt(summary),
    }),
    update: () => ({
        name: opt(name),
        details: opt(details),
        summary: opt(summary),
    }),
});

export const apiVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        callLink: opt(url(d)),
        documentationLink: opt(url(d)),
        isPrivate: opt(bool),
        schemaLanguage: opt(schemaLanguage),
        schemaText: opt(schemaText),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "opt", apiValidation, ["versions"]],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", apiVersionTranslationValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        callLink: opt(url(d)),
        documentationLink: opt(url(d)),
        isPrivate: opt(bool),
        schemaLanguage: opt(schemaLanguage),
        schemaText: opt(schemaText),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Update"], "one", "opt", apiValidation, ["versions"]],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", apiVersionTranslationValidation],
    ], [], d),
};
