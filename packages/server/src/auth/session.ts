import { Session, SessionUser } from '@shared/consts';
import { getActiveFocusMode } from '@shared/utils';
import { Request } from "express";
import { CustomError, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events";
import { PrismaType, SessionUserToken } from "../types";
import { getUser } from './request';

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
                }
            }
        }
    },
    labels: {
        select: {
            id: true,
            label: {
                select: {
                    id: true,
                    color: true,
                    label: true,
                }
            }
        }
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
                        }
                    }
                }
            }
        }
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
                        }
                    }
                }
            },
            translations: {
                select: {
                    id: true,
                    description: true,
                    language: true,
                    name: true,
                }
            }
        }
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
                }
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
                }
            }
        }
    }
})

/**
 * Creates SessionUser object from user.
 * Also updates user's lastSessionVerified time
 * @param user User object
 * @param prisma Prisma type
 * @param req Express request object
 */
export const toSessionUser = async (user: { id: string }, prisma: PrismaType, req: Partial<Request>): Promise<SessionUser> => {
    if (!user.id)
        throw new CustomError('0064', 'NotFound', req.languages ?? ['en']);
    // Create time frame to find schedule data for. 
    const now = new Date();
    const startDate = now;
    const endDate = new Date(now.setDate(now.getDate() + 7));
    // Update user's lastSessionVerified, and query for user data
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
                        }
                    }
                }
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
                }
            }
        }
    })
    // Find active focus mode
    const currentActiveFocusMode = getUser(req)?.activeFocusMode;
    const currentModeData = (userData.focusModes as any).find((fm: any) => fm.id === currentActiveFocusMode?.mode?.id);
    const activeFocusMode = getActiveFocusMode(
        currentModeData ? {
            ...currentActiveFocusMode,
            mode: currentModeData,
        } as any : undefined,
        (userData.focusModes as any) ?? []
    );
    // Calculate langugages, by combining user's languages with languages 
    // in request. Make sure to remove duplicates
    let languages: string[] = userData.languages.map((l) => l.language).filter(Boolean) as string[];
    if (req.languages) languages.push(...req.languages);
    languages = [...new Set(languages)];
    // Return shaped SessionUser object
    const result = {
        __typename: 'SessionUser' as const,
        activeFocusMode,
        apisCount: userData._count?.apis ?? 0,
        bookmarkLists: userData.bookmakLists.map(({ _count, ...rest }) => ({
            ...rest,
            bookmarksCount: _count?.bookmarks ?? 0,
        })) as any[],
        focusModes: userData.focusModes as any[],
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
    }
    return result;
}

/**
 * Converts SessionUserToken object o SessionUser object, 
 * by adding dummy fields. Useful for avoiding type errors
 * @param user SessionUserToken object
 * @returns SessionUser object
 */
export const sessionUserTokenToUser = (user: SessionUserToken): SessionUser => ({
    __typename: 'SessionUser' as const,
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
        __typename: 'ActiveFocusMode' as const,
        ...user.activeFocusMode,
        mode: {
            __typename: 'FocusMode' as const,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            name: '',
            description: undefined,
            filters: [],
            labels: [],
            reminderList: undefined,
            resourceList: undefined,
            schedule: undefined,
            ...user.activeFocusMode.mode,
        }
    } : undefined,
})

/**
 * Creates session object from user and existing session data
 * @param user User object
 * @param prisma 
 * @param req Express request object (with current session data)
 * @returns Updated session object, with user data added to the START of the users array
 */
export const toSession = async (user: { id: string }, prisma: PrismaType, req: Partial<Request>): Promise<Session> => {
    const sessionUser = await toSessionUser(user, prisma, req);
    return {
        __typename: 'Session' as const,
        isLoggedIn: true,
        // Make sure users are unique by id
        users: [sessionUser, ...(req.users ?? []).filter((u: SessionUserToken) => u.id !== sessionUser.id).map(sessionUserTokenToUser)],
    }
}