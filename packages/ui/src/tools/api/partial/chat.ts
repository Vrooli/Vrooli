import { Chat, ChatTranslation, ChatYou } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const chatTranslation: GqlPartial<ChatTranslation> = {
    __typename: 'ChatTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const chatYou: GqlPartial<ChatYou> = {
    __typename: 'ChatYou',
    common: {
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
    full: {},
    list: {},
}

export const chat: GqlPartial<Chat> = {
    __typename: 'Chat',
    common: {
        id: true,
        openToAnyoneWithInvite: true,
        organization: async () => rel((await import('./organization')).organization, 'nav'),
        restrictedToRoles: async () => rel((await import('./role')).role, 'full'),
        participantsCount: true,
        invitesCount: true,
        you: () => rel(chatYou, 'full'),
    },
    full: {
        __define: {
            0: async () => rel((await import('./label')).label, 'full'),
        },
        participants: async () => rel((await import('./user')).user, 'nav'),
        invites: async () => rel((await import('./chatInvite')).chatInvite, 'list', { omit: 'chat' }),
        messages: async () => rel((await import('./chatMessage')).chatMessage, 'list', { omit: 'chat' }),
        labels: { __use: 0 },
        translations: () => rel(chatTranslation, 'full'),
    },
    list: {
        __define: {
            0: async () => rel((await import('./label')).label, 'list'),
        },
        labels: { __use: 0 },
        translations: () => rel(chatTranslation, 'list'),
    }
}