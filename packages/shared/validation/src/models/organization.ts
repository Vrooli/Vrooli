import { bio, bool, handle, id, name, opt, req, transRel, YupModel, yupObj } from '../utils';
import { tagValidation } from './tag';
import { resourceListValidation } from './resourceList';
import { roleValidation } from './role';
import { memberInviteValidation } from './memberInvite';

export const organizationTranslationValidation: YupModel = transRel({
    create: {
        bio: opt(bio),
        name: req(name),
    },
    update: {
        bio: opt(bio),
        name: opt(name),
    },
})

export const organizationValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
    }, [
        ['resourceList', ['Create'], 'one', 'opt', resourceListValidation],
        ['tags', ['Connect', 'Create'], 'many', 'opt', tagValidation],
        ['roles', ['Create'], 'many', 'opt', roleValidation],
        ['memberInvites', ['Create'], 'many', 'opt', memberInviteValidation],
        ['translations', ['Create'], 'many', 'opt', organizationTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isOpenToNewMembers: opt(bool),
        isPrivate: opt(bool),
    }, [
        ['resourceList', ['Update'], 'one', 'opt', resourceListValidation],
        ['tags', ['Connect', 'Disconnect', 'Create'], 'many', 'opt', tagValidation],
        ['roles', ['Create', 'Update', 'Delete'], 'many', 'opt', roleValidation],
        ['memberInvites', ['Create', 'Delete'], 'many', 'opt', memberInviteValidation],
        ['members', ['Delete'], 'many', 'opt'],
    ], [], o),
}