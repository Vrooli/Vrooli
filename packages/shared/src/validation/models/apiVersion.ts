import * as yup from "yup";
import { bool, details, id, maxStrErr, minStrErr, name, opt, req, summary, transRel, url, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { apiValidation } from "./api";
import { resourceListValidation } from "./resourceList";

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
