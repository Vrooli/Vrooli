import { BookmarkFor } from '@shared/consts';
import { enumToYup, id, req, YupModel, yupObj } from "../utils";
import { bookmarkListValidation } from './bookmarkList';

const bookmarkFor = enumToYup(BookmarkFor);

export const bookmarkValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        bookmarkFor: req(bookmarkFor),
    }, [
        ['for', ['Connect'], 'one', 'req'],
        ['list', ['Connect', 'Create'], 'many', 'opt', bookmarkListValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['list', ['Connect', 'Update'], 'many', 'opt', bookmarkListValidation],
    ], [], o),
}