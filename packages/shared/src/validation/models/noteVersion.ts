import * as yup from "yup";
import { opt, req, reqArr } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, intPositiveOrZero, language, name, versionLabel, versionNotes } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { noteValidation } from "./note.js";

const text = yup.string().trim().max(65536, maxStrErr);

export const notePageValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        pageIndex: req(intPositiveOrZero),
        text: text.defined().strict(true),
    }, [], [], d),
    update: (d) => yupObj({
        id: req(id),
        pageIndex: opt(intPositiveOrZero),
        text: text.defined().strict(true).optional(),
    }, [], [], d),
};

// We define "read" for this validation so that we can use it for translation validations. 
// Other translations don't need it because they don't have relations. In the UI, we need 
// to validate a translation object with a "pages" relation, and not "pagesCreate" or "pagesUpdate"
export const noteVersionTranslationValidation: YupModel<["create", "update", "read"]> = {
    create: (d) => yupObj({
        id: req(id),
        language: req(language),
        description: opt(description),
        name: opt(name),
    }, [
        ["pages", ["Create", "Update", "Delete"], "many", "opt", notePageValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        language: opt(language),
        description: opt(description),
        name: opt(name),
    }, [
        ["pages", ["Create", "Update", "Delete"], "many", "opt", notePageValidation],
    ], [], d),
    read: (d) => yupObj({
        id: req(id),
        language: req(language),
        description: opt(description),
        name: opt(name),
        pages: reqArr(notePageValidation.create(d)),
    }, [], [], d),
};

export const noteVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "opt", noteValidation, ["versions"]],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", noteVersionTranslationValidation],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Update"], "one", "opt", noteValidation, ["versions"]],
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", noteVersionTranslationValidation],
    ], [], d),
};
