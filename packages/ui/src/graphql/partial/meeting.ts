import { Meeting, MeetingTranslation, MeetingYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { meetingInvitePartial } from "./meetingInvite";
import { organizationPartial } from "./organization";
import { rolePartial } from "./role";
import { userPartial } from "./user";

export const meetingTranslationPartial: GqlPartial<MeetingTranslation> = {
    __typename: 'MeetingTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        link: true,
        name: true,
    },
}

export const meetingYouPartial: GqlPartial<MeetingYou> = {
    __typename: 'MeetingYou',
    full: {
        canDelete: true,
        canEdit: true,
        canInvite: true,
    },
}

export const meetingPartial: GqlPartial<Meeting> = {
    __typename: 'Meeting',
    common: {
        id: true,
        openToAnyoneWithInvite: true,
        showOnOrganizationProfile: true,
        timeZone: true,
        eventStart: true,
        eventEnd: true,
        recurring: true,
        recurrStart: true,
        recurrEnd: true,
        organization: () => relPartial(organizationPartial, 'nav'),
        restrictedToRoles: () => relPartial(rolePartial, 'list'),
        attendeesCount: true,
        invitesCount: true,
        you: () => relPartial(meetingYouPartial, 'full'),
    },
    full: {
        attendees: () => relPartial(userPartial, 'nav'),
        invites: () => relPartial(meetingInvitePartial, 'list', { omit: 'meeting' }),
        labels: () => relPartial(labelPartial, 'full'),
        translations: () => relPartial(meetingTranslationPartial, 'full'),
    },
    list: {
        labels: () => relPartial(labelPartial, 'list'),
        translations: () => relPartial(meetingTranslationPartial, 'list'),
    }
}