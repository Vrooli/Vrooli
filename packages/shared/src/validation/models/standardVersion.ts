import { InputType } from "@local/shared";
import * as yup from "yup";
import { bool, description, enumToYup, id, jsonVariable, maxStrErr, name, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { resourceListValidation } from "./resourceList";
import { standardValidation } from "./standard";

const standardDefault = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
const standardType = enumToYup(InputType);
const standardProps = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
const standardYup = yup.string().trim().removeEmptyString().max(8192, maxStrErr);

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
        isComplete: opt(bool),
        isFile: opt(bool),
        isPrivate: opt(bool),
        default: opt(standardDefault),
        standardType: req(standardType),
        props: req(standardProps),
        yup: opt(standardYup),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect"], "many", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["root", ["Connect", "Create"], "one", "req", standardValidation, ["versions"]],
        ["translations", ["Create"], "many", "opt", standardVersionTranslationValidation],
    ], [["rootConnect", "rootCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isFile: opt(bool),
        isPrivate: opt(bool),
        default: opt(standardDefault),
        standardType: opt(standardType),
        props: opt(standardProps),
        yup: opt(standardYup),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", standardValidation, ["versions"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", standardVersionTranslationValidation],
    ], [], d),
};
