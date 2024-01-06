import { bio, bool, handle, id, imageFile, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { memberInviteValidation } from "./memberInvite";
import { resourceListValidation } from "./resourceList";
import { roleValidation } from "./role";
import { tagValidation } from "./tag";

export const organizationTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        bio: opt(bio),
        name: req(name),
    }),
    update: () => ({
        bio: opt(bio),
        name: opt(name),
    }),
});

export const organizationValidation: YupModel<["create", "update"]> = {
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
        ["translations", ["Create"], "many", "opt", organizationTranslationValidation],
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
