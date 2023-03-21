import { Meeting, MeetingCreateInput, MeetingTranslation, MeetingTranslationCreateInput, MeetingTranslationUpdateInput, MeetingUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { MeetingInviteShape, shapeMeetingInvite } from "./meetingInvite";
import { ScheduleShape } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type MeetingTranslationShape = Pick<MeetingTranslation, 'id' | 'language' | 'description' | 'link' | 'name'> & {
    __typename?: 'MeetingTranslation';
}

export type MeetingShape = Pick<Meeting, 'id' | 'openToAnyoneWithInvite' | 'showOnOrganizationProfile'> & {
    __typename?: 'Meeting';
    organization: { id: string };
    restrictedToRoles?: { id: string }[] | null;
    invites?: MeetingInviteShape[] | null;
    labels?: { id: string }[] | null;
    schedule?: ScheduleShape | null;
    translations?: MeetingTranslationShape[] | null;
}

export const shapeMeetingTranslation: ShapeModel<MeetingTranslationShape, MeetingTranslationCreateInput, MeetingTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'link', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'link', 'name'), a)
}

export const shapeMeeting: ShapeModel<MeetingShape, MeetingCreateInput, MeetingUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'openToAnyoneWithInvite', 'showOnOrganizationProfile'),
        ...createRel(d, 'organization', ['Connect'], 'one'),
        ...createRel(d, 'restrictedToRoles', ['Connect'], 'many'),
        ...createRel(d, 'invites', ['Create'], 'many', shapeMeetingInvite, (i) => ({ ...i, meeting: { id: d.id } })),
        ...createRel(d, 'labels', ['Connect'], 'many'),
        ...createRel(d, 'schedule', ['Create'], 'one', shapeSchedule),
        ...createRel(d, 'translations', ['Create'], 'many', shapeMeetingTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'openToAnyoneWithInvite', 'showOnOrganizationProfile'),
        ...updateRel(o, u, 'restrictedToRoles', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'invites', ['Create', 'Update', 'Delete'], 'many', shapeMeetingInvite, (i) => ({ ...i, meeting: { id: o.id } })),
        ...updateRel(o, u, 'labels', ['Connect', 'Disconnect'], 'many'),
        ...createRel(d, 'schedule', ['Create', 'Update'], 'one', shapeSchedule),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeMeetingTranslation),
    }, a)
}