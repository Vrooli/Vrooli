import * as yup from "yup";
import { bool, description, id, intPositiveOrZero, language, maxStrErr, name, opt, req, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { noteValidation } from "./note";

const text = yup.string().trim().max(65536, maxStrErr);

export const notePageValidation: YupModel = {
    create: (d) => yupObj({
        pageIndex: req(intPositiveOrZero),
        text: text.defined().strict(true),
    }, [], [], d),
    update: (d) => yupObj({
        pageIndex: opt(intPositiveOrZero),
        text: text.defined().strict(true).optional(),
    }, [], [], d),
};

export const noteVersionTranslationValidation: YupModel = {
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
};

export const noteVersionValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "req", noteValidation, ["versions"]],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", noteVersionTranslationValidation],
    ], [["rootConnect", "rootCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", noteVersionTranslationValidation],
    ], [], d),
};
