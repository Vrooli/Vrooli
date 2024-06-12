import { ActiveFocusMode, getActiveFocusMode, Session, SessionUser } from "@local/shared";
import { Request } from "express";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events/schedule";
import { SessionData, SessionUserToken } from "../types";
import { getUser } from "./request";

export const focusModeSelect = (startDate: Date, endDate: Date) => ({
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

/**
 * Creates SessionUser object from user.
 * @param user User object
 * @param req Express request object
 */
export const toSessionUser = async (user: { id: string }, req: Partial<Request>): Promise<SessionUser> => {
    if (!user.id)
        throw new CustomError("0064", "NotFound", req.session?.languages ?? ["en"]);
    // Create time frame to find schedule data for. 
    const now = new Date();
    const startDate = now;
    const endDate = new Date(now.setDate(now.getDate() + 7));
    // Query for user data
    const userData = await prismaInstance.user.findUnique({
        where: { id: user.id },
        select: {
            id: true,
            updated_at: true,
            handle: true,
            languages: { select: { language: true } },
            name: true,
            theme: true,
            premium: { select: { id: true, credits: true, expiresAt: true } },
            profileImage: true,
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
                    codes: true,
                    notes: true,
                    memberships: true,
                    projects: true,
                    questionsAsked: true,
                    routines: true,
                    standards: true,
                },
            },
        },
    });
    if (!userData)
        throw new CustomError("0510", "NotFound", req.session?.languages ?? ["en"]);
    // Find active focus mode
    const focusModesWithSupp = (userData.focusModes as object[])?.map((fm: any) => ({
        ...fm,
        you: {
            __typename: "FocusModeYou" as const,
            canDelete: true,
            canRead: true,
            canUpdate: true,
        },
    })) as SessionUser["focusModes"];
    const currentActiveFocusMode = getUser(req.session as SessionData)?.activeFocusMode;
    const currentModeData = focusModesWithSupp.find((fm) => fm.id === currentActiveFocusMode?.mode?.id);
    const activeFocusMode = await getActiveFocusMode(
        currentModeData ? {
            ...currentActiveFocusMode,
            mode: currentModeData,
        } as ActiveFocusMode : undefined,
        focusModesWithSupp ?? [],
    );
    // Calculate langugages, by combining user's languages with languages 
    // in request. Make sure to remove duplicates
    let languages: string[] = userData.languages.map((l) => l.language).filter(Boolean) as string[];
    if (req.session?.languages) languages.push(...req.session.languages);
    languages = [...new Set(languages)];
    // Return shaped SessionUser object
    const result = {
        __typename: "SessionUser" as const,
        activeFocusMode,
        apisCount: userData._count?.apis ?? 0,
        bookmarkLists: userData.bookmakLists.map(({ _count, ...rest }) => ({
            ...rest,
            bookmarksCount: _count?.bookmarks ?? 0,
        })) as SessionUser["bookmarkLists"],
        codesCount: userData._count?.codes ?? 0,
        credits: (userData.premium?.credits ?? BigInt(0)) + "", // Convert to string because BigInt can't be serialized
        focusModes: focusModesWithSupp,
        handle: userData.handle ?? undefined,
        hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
        id: user.id,
        languages,
        membershipsCount: userData._count?.memberships ?? 0,
        name: userData.name,
        notesCount: userData._count?.notes ?? 0,
        profileImage: userData.profileImage,
        projectsCount: userData._count?.projects ?? 0,
        questionsAskedCount: userData._count?.questionsAsked ?? 0,
        routinesCount: userData._count?.routines ?? 0,
        standardsCount: userData._count?.standards ?? 0,
        theme: userData.theme,
        updated_at: userData.updated_at,
    };
    return result;
};

/**
 * Converts SessionUserToken object o SessionUser object, 
 * by adding dummy fields. Useful for avoiding type errors
 * @param user SessionUserToken object
 * @returns SessionUser object
 */
export const sessionUserTokenToUser = (user: SessionUserToken): SessionUser => ({
    __typename: "SessionUser" as const,
    apisCount: 0,
    bookmarkLists: [],
    codesCount: 0,
    focusModes: [],
    membershipsCount: 0,
    notesCount: 0,
    projectsCount: 0,
    questionsAskedCount: 0,
    routinesCount: 0,
    standardsCount: 0,
    ...user,
    activeFocusMode: user.activeFocusMode ? {
        __typename: "ActiveFocusMode" as const,
        ...user.activeFocusMode,
        mode: {
            __typename: "FocusMode" as const,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            name: "",
            description: undefined,
            filters: [],
            labels: [],
            reminderList: user.activeFocusMode?.mode?.reminderList ? {
                __typename: "ReminderList" as const,
                ...user.activeFocusMode.mode.reminderList,
            } as any : undefined,
            resourceList: undefined,
            schedule: undefined,
            you: {
                __typename: "FocusModeYou" as const,
                canDelete: true,
                canRead: true,
                canUpdate: true,
            },
            ...user.activeFocusMode.mode,
        },
    } : undefined,
});

/**
 * Creates session object from user and existing session data
 * @param user User object
 * @param req Express request object (with current session data)
 * @returns Updated session object, with user data added to the START of the users array
 */
export const toSession = async (user: { id: string }, req: Partial<Request>): Promise<Session> => {
    const sessionUser = await toSessionUser(user, req);
    return {
        __typename: "Session" as const,
        isLoggedIn: true,
        // Make sure users are unique by id
        users: [sessionUser, ...(req.session?.users ?? []).filter((u: SessionUserToken) => u.id !== sessionUser.id).map(sessionUserTokenToUser)],
    };
};
