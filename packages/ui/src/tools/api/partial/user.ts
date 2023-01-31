import { User, UserTranslation, UserYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const userTranslationPartial: GqlPartial<UserTranslation> = {
    __typename: 'UserTranslation',
    common: {
        id: true,
        language: true,
        bio: true,
    },
    full: {},
    list: {},
}

export const userYouPartial: GqlPartial<UserYou> = {
    __typename: 'UserYou',
    common: {
        canDelete: true,
        canEdit: true,
        canReport: true,
        isStarred: true,
        isViewed: true,
    },
    full: {},
    list: {},
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
        stats: async () => relPartial((await import('./statsUser')).statsUserPartial, 'full'),
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
    common: {
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
        emails: async () => relPartial((await import('./email')).emailPartial, 'full'),
        pushDevices: async () => relPartial((await import('./pushDevice')).pushDevicePartial, 'full'),
        wallets: async () => relPartial((await import('./wallet')).walletPartial, 'common'),
        notifications: async () => relPartial((await import('./notification')).notificationPartial, 'full'),
        notificationSettings: true,
        translations: () => relPartial(userTranslationPartial, 'full'),
        schedules: async () => relPartial((await import('./userSchedule')).userSchedulePartial, 'full'),
        stats: async () => relPartial((await import('./statsUser')).statsUserPartial, 'full'),
    },
    full: {},
    list: {},
}