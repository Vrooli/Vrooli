import { getActiveFocusMode } from "@local/utils";
import { CustomError, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events";
import { getUser } from "./request";
export const focusModeSelect = (startDate, endDate) => ({
    id: true,
    name: true,
    description: true,
    filters: {
        select: {
            id: true,
            filterType: true,
            tag: {
                select: {
                    id: true,
                    tag: true,
                },
            },
        },
    },
    labels: {
        select: {
            id: true,
            label: {
                select: {
                    id: true,
                    color: true,
                    label: true,
                },
            },
        },
    },
    reminderList: {
        select: {
            id: true,
            created_at: true,
            updated_at: true,
            reminders: {
                select: {
                    id: true,
                    created_at: true,
                    updated_at: true,
                    name: true,
                    description: true,
                    dueDate: true,
                    index: true,
                    isComplete: true,
                    reminderItems: {
                        select: {
                            id: true,
                            created_at: true,
                            updated_at: true,
                            name: true,
                            description: true,
                            dueDate: true,
                            index: true,
                            isComplete: true,
                        },
                    },
                },
            },
        },
    },
    resourceList: {
        select: {
            id: true,
            created_at: true,
            updated_at: true,
            resources: {
                select: {
                    id: true,
                    created_at: true,
                    updated_at: true,
                    index: true,
                    link: true,
                    usedFor: true,
                    translations: {
                        select: {
                            id: true,
                            description: true,
                            language: true,
                            name: true,
                        },
                    },
                },
            },
            translations: {
                select: {
                    id: true,
                    description: true,
                    language: true,
                    name: true,
                },
            },
        },
    },
    schedule: {
        select: {
            id: true,
            created_at: true,
            updated_at: true,
            startTime: true,
            endTime: true,
            timezone: true,
            exceptions: {
                where: scheduleExceptionsWhereInTimeframe(startDate, endDate),
                select: {
                    id: true,
                    originalStartTime: true,
                    newStartTime: true,
                    newEndTime: true,
                },
            },
            recurrences: {
                where: scheduleRecurrencesWhereInTimeframe(startDate, endDate),
                select: {
                    id: true,
                    recurrenceType: true,
                    interval: true,
                    dayOfWeek: true,
                    dayOfMonth: true,
                    month: true,
                    endDate: true,
                },
            },
        },
    },
});
export const toSessionUser = async (user, prisma, req) => {
    if (!user.id)
        throw new CustomError("0064", "NotFound", req.languages ?? ["en"]);
    const now = new Date();
    const startDate = now;
    const endDate = new Date(now.setDate(now.getDate() + 7));
    const userData = await prisma.user.update({
        where: { id: user.id },
        data: { lastSessionVerified: new Date().toISOString() },
        select: {
            id: true,
            handle: true,
            languages: { select: { language: true } },
            name: true,
            theme: true,
            premium: { select: { id: true, expiresAt: true } },
            bookmakLists: {
                select: {
                    id: true,
                    created_at: true,
                    updated_at: true,
                    label: true,
                    _count: {
                        select: {
                            bookmarks: true,
                        },
                    },
                },
            },
            focusModes: { select: focusModeSelect(startDate, endDate) },
            _count: {
                select: {
                    apis: true,
                    notes: true,
                    memberships: true,
                    projects: true,
                    questionsAsked: true,
                    routines: true,
                    smartContracts: true,
                    standards: true,
                },
            },
        },
    });
    const currentActiveFocusMode = getUser(req)?.activeFocusMode;
    const currentModeData = userData.focusModes.find((fm) => fm.id === currentActiveFocusMode?.mode?.id);
    const activeFocusMode = getActiveFocusMode(currentModeData ? {
        ...currentActiveFocusMode,
        mode: currentModeData,
    } : undefined, userData.focusModes ?? []);
    let languages = userData.languages.map((l) => l.language).filter(Boolean);
    if (req.languages)
        languages.push(...req.languages);
    languages = [...new Set(languages)];
    const result = {
        __typename: "SessionUser",
        activeFocusMode,
        apisCount: userData._count?.apis ?? 0,
        bookmarkLists: userData.bookmakLists.map(({ _count, ...rest }) => ({
            ...rest,
            bookmarksCount: _count?.bookmarks ?? 0,
        })),
        focusModes: userData.focusModes,
        handle: userData.handle ?? undefined,
        hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
        id: user.id,
        languages,
        membershipsCount: userData._count?.memberships ?? 0,
        name: userData.name,
        notesCount: userData._count?.notes ?? 0,
        projectsCount: userData._count?.projects ?? 0,
        questionsAskedCount: userData._count?.questionsAsked ?? 0,
        routinesCount: userData._count?.routines ?? 0,
        smartContractsCount: userData._count?.smartContracts ?? 0,
        standardsCount: userData._count?.standards ?? 0,
        theme: userData.theme,
    };
    return result;
};
export const sessionUserTokenToUser = (user) => ({
    __typename: "SessionUser",
    apisCount: 0,
    bookmarkLists: [],
    focusModes: [],
    membershipsCount: 0,
    notesCount: 0,
    projectsCount: 0,
    questionsAskedCount: 0,
    routinesCount: 0,
    smartContractsCount: 0,
    standardsCount: 0,
    ...user,
    activeFocusMode: user.activeFocusMode ? {
        __typename: "ActiveFocusMode",
        ...user.activeFocusMode,
        mode: {
            __typename: "FocusMode",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            name: "",
            description: undefined,
            filters: [],
            labels: [],
            reminderList: undefined,
            resourceList: undefined,
            schedule: undefined,
            ...user.activeFocusMode.mode,
        },
    } : undefined,
});
export const toSession = async (user, prisma, req) => {
    const sessionUser = await toSessionUser(user, prisma, req);
    return {
        __typename: "Session",
        isLoggedIn: true,
        users: [sessionUser, ...(req.users ?? []).filter((u) => u.id !== sessionUser.id).map(sessionUserTokenToUser)],
    };
};
//# sourceMappingURL=session.js.map