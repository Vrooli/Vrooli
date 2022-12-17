import { bio, handle, id, name, opt, rel, req, transRel, YupModel } from '../utils';
import * as yup from 'yup';
import { tagValidation } from './tag';
import { resourceListValidation } from './resourceList';
import { roleValidation } from './role';
import { memberInviteValidation } from './memberInvite';

const isOpenToNewMembers = yup.boolean()
const isPrivate = yup.boolean()

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
    create: yup.object().shape({
        id: req(id),
        handle: opt(handle),
        isOpenToNewMembers: opt(isOpenToNewMembers),
        isPrivate: opt(isPrivate),
        ...rel('resourceList', ['Create'], 'one', 'opt', resourceListValidation),
        ...rel('tags', ['Connect', 'Create'], 'many', 'opt', tagValidation),
        ...rel('roles', ['Create'], 'many', 'opt', roleValidation),
        ...rel('memberInvites', ['Create'], 'many', 'opt', memberInviteValidation),
        ...rel('translations', ['Create'], 'many', 'opt', organizationTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        handle: opt(handle),
        isOpenToNewMembers: opt(isOpenToNewMembers),
        isPrivate: opt(isPrivate),
        ...rel('resourceList', ['Update'], 'one', 'opt', resourceListValidation),
        ...rel('tags', ['Connect', 'Disconnect', 'Create'], 'many', 'opt', tagValidation),
        ...rel('roles', ['Create', 'Update', 'Delete'], 'many', 'opt', roleValidation),
        ...rel('memberInvites', ['Create', 'Delete'], 'many', 'opt', memberInviteValidation),
        ...rel('members', ['Delete'], 'many', 'opt'),
    }),
}