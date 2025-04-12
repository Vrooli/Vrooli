import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bio, bool, handle, id, imageFile, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { memberInviteValidation } from "./memberInvite.js";
import { resourceListValidation } from "./resourceList.js";
import { roleValidation } from "./role.js";
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
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
        profileImage: opt(imageFile),
    }, [
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["roles", ["Create"], "many", "opt", roleValidation],
        ["memberInvites", ["Create"], "many", "opt", memberInviteValidation],
        ["translations", ["Create"], "many", "opt", teamTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        bannerImage: opt(imageFile),
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
        profileImage: opt(imageFile),
    }, [
        ["resourceList", ["Update"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Disconnect", "Create"], "many", "opt", tagValidation],
        ["roles", ["Create", "Update", "Delete"], "many", "opt", roleValidation],
        ["memberInvites", ["Create", "Delete"], "many", "opt", memberInviteValidation],
        ["members", ["Delete"], "many", "opt"],
    ], [], d),
};
