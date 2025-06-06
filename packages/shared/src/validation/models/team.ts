/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in team.test.ts
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bio, bool, config, handle, id, imageFile, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { memberInviteValidation } from "./memberInvite.js";
import { tagValidation } from "./tag.js";

export const teamTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        bio: opt(bio),
        name: req(name),
    }),
    update: () => ({
        bio: opt(bio),
        name: opt(name),
    }),
});

export const teamValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        bannerImage: opt(imageFile),
        config: opt(config),
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
        profileImage: opt(imageFile),
    }, [
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["memberInvites", ["Create"], "many", "opt", memberInviteValidation],
        ["translations", ["Create"], "many", "opt", teamTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        bannerImage: opt(imageFile),
        config: opt(config),
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
        profileImage: opt(imageFile),
    }, [
        ["tags", ["Connect", "Disconnect", "Create"], "many", "opt", tagValidation],
        ["memberInvites", ["Create", "Delete"], "many", "opt", memberInviteValidation],
        ["members", ["Delete"], "many", "opt"],
    ], [], d),
};
/* c8 ignore stop */
