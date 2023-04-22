import * as yup from "yup";
import { blankToUndefined, id, maxStrErr, opt, req, YupModel, yupObj } from "../utils";
import { bookmarkValidation } from "./bookmark";

const label = yup.string().transform(blankToUndefined).max(128, maxStrErr);

export const bookmarkListValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        label: req(label),
    }, [
        ["bookmarks", ["Connect", "Create"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        label: opt(label),
    }, [
        ["bookmarks", ["Connect", "Create", "Update", "Delete"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], o),
};
