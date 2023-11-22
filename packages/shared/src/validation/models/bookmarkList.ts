import * as yup from "yup";
import { id, maxStrErr, opt, req, YupModel, yupObj } from "../utils";
import { bookmarkValidation } from "./bookmark";

const label = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const bookmarkListValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        label: req(label),
    }, [
        ["bookmarks", ["Connect", "Create"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        label: opt(label),
    }, [
        ["bookmarks", ["Connect", "Create", "Update", "Delete"], "many", "opt", bookmarkValidation, ["list"]],
    ], [], d),
};
