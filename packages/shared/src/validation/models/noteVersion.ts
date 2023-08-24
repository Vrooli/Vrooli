import * as yup from "yup";
import { blankToUndefined, bool, description, id, intPositiveOrZero, language, maxStrErr, name, opt, req, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { noteValidation } from "./note";

const text = yup.string().transform(blankToUndefined).max(65536, maxStrErr);

export const notePageValidation: YupModel = {
    create: ({ o }) => yupObj({
        pageIndex: req(intPositiveOrZero),
        text: req(text),
    }, [], [], o),
    update: ({ o }) => yupObj({
        pageIndex: opt(intPositiveOrZero),
        text: opt(text),
    }, [], [], o),
};

export const noteVersionTranslationValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        language: req(language),
        description: opt(description),
        name: opt(name),
    }, [
        ["pages", ["Create", "Update", "Delete"], "many", "opt", notePageValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        language: opt(language),
        description: opt(description),
        name: opt(name),
    }, [
        ["pages", ["Create", "Update", "Delete"], "many", "opt", notePageValidation],
    ], [], o),
};

export const noteVersionValidation: YupModel = {
    create: ({ o, minVersion = "0.0.1" }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "req", noteValidation, ["versions"]],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", noteVersionTranslationValidation],
    ], [["rootConnect", "rootCreate"]], o),
    update: ({ o, minVersion = "0.0.1" }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", noteVersionTranslationValidation],
    ], [], o),
};
