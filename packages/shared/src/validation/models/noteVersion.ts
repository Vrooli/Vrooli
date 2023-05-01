import * as yup from "yup";
import { blankToUndefined, bool, description, id, maxStrErr, name, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { noteValidation } from "./note";

const text = yup.string().transform(blankToUndefined).max(65536, maxStrErr);

export const noteVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: opt(name),
        text: req(text),
    },
    update: {
        description: opt(description),
        name: opt(name),
        text: opt(text),
    },
});

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
