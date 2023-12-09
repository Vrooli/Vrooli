import { bool, details, id, name, opt, req, summary, transRel, url, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { apiValidation } from "./api";
import { resourceListValidation } from "./resourceList";

export const apiVersionTranslationValidation: YupModel = transRel({
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

export const apiVersionValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        callLink: opt(url(d)),
        documentationLink: opt(url(d)),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "req", apiValidation, ["versions"]],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", apiVersionTranslationValidation],
    ], [["rootConnect", "rootCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
        callLink: opt(url(d)),
        documentationLink: opt(url(d)),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", apiVersionTranslationValidation],
    ], [], d),
};
