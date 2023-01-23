import { User, UserTranslation, UserYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const userTranslationPartial: GqlPartial<UserTranslation> = {
    __typename: 'UserTranslation',
    full: {
        id: true,
        language: true,
        bio: true,
    },
}

export const userYouPartial: GqlPartial<UserYou> = {
    __typename: 'UserYou',
    full: {
        canDelete: true,
        canEdit: true,
        canReport: true,
        isStarred: true,
        isViewed: true,
    },
}

export const userPartial: GqlPartial<User> = {
    __typename: 'User',
    common: {
        id: true,
        created_at: true,
        handle: true,
        name: true,
        stars: true,
        reportsCount: true,
        you: () => relPartial(userYouPartial, 'full'),
    },
    full: {
        stats: () => relPartial(require('./statsUser').statsUserPartial, 'full'),
        translations: () => relPartial(userTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(userTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        name: true,
        handle: true,
    }
}

export const profilePartial: GqlPartial<User> = {
    __typename: 'User',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        handle: true,
        isPrivate: true,
        isPrivateApis: true,
        isPrivateApisCreated: true,
        isPrivateMemberships: true,
        isPrivateOrganizationsCreated: true,
        isPrivateProjects: true,
        isPrivateProjectsCreated: true,
        isPrivatePullRequests: true,
        isPrivateQuestionsAnswered: true,
        isPrivateQuestionsAsked: true,
        isPrivateQuizzesCreated: true,
        isPrivateRoles: true,
        isPrivateRoutines: true,
        isPrivateRoutinesCreated: true,
        isPrivateStandards: true,
        isPrivateStandardsCreated: true,
        isPrivateStars: true,
        isPrivateVotes: true,
        name: true,
        theme: true,
        emails: () => relPartial(require('./email').emailPartial, 'full'),
        pushDevices: () => relPartial(require('./pushDevice').pushDevicePartial, 'full'),
        wallets: () => relPartial(require('./wallet').walletPartial, 'full'),
        notifications: () => relPartial(require('./notification').notificationPartial, 'full'),
        notificationSettings: true,
        translations: () => relPartial(userTranslationPartial, 'full'),
        schedules: () => relPartial(require('./userSchedule').userSchedulePartial, 'full'),
        stats: () => relPartial(require('./statsUser').statsUserPartial, 'full'),
    },
}