import { ActiveFocusMode, getActiveFocusMode, Session, SessionUser, uuidValidate, WEEKS_1_DAYS } from "@local/shared";
import { Request } from "express";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events/schedule";
import { RecursivePartialNullable, SessionData, SessionUserToken } from "../types";

export function focusModeSelect(startDate: Date, endDate: Date) {
    return {
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
    } as const;
}

export class SessionService {
    private static instance: SessionService;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): SessionService {
        if (!SessionService.instance) {
            SessionService.instance = new SessionService();
        }
        return SessionService.instance;
    }

    /**
    * Finds current user in Request object. Also validates that the user data is valid
    * @param session The Request's session property
    * @returns First userId in Session object, or null if not found/invalid
    */
    static getUser(session: RecursivePartialNullable<Pick<SessionData, "users">>): SessionUserToken | null {
        if (!session || !Array.isArray(session?.users) || session.users.length === 0) return null;
        const user = session.users[0];
        return user !== undefined && typeof user.id === "string" && uuidValidate(user.id) ? user : null;
    }

    /**
     * Ensures that any erroneous data is stripped from the user session data.
     * @param user The user data to be cleaned
     * @returns The cleaned user data
     */
    static stripSessionUserToken(user: SessionUserToken): SessionUserToken {
        const { activeFocusMode, credits, handle, hasPremium, id, languages, name, profileImage, session, updated_at } = user;
        return {
            id,
            credits,
            handle: handle ?? undefined,
            hasPremium: hasPremium ?? false,
            languages: languages ?? [],
            name: name ?? undefined,
            profileImage: profileImage ?? undefined,
            updated_at: updated_at,
            activeFocusMode: activeFocusMode ? {
                mode: {
                    id: activeFocusMode.mode?.id,
                    reminderList: activeFocusMode?.mode?.reminderList ? {
                        id: activeFocusMode.mode.reminderList.id,
                    } : undefined,
                },
                stopCondition: activeFocusMode.stopCondition,
                stopTime: activeFocusMode.stopTime,
            } : undefined,
            session: {
                id: session.id,
                lastRefreshAt: session.lastRefreshAt,
            },
        };
    }

    /**
     * Creates SessionUser object from user.
     * @param userId User ID
     * @param sessionId Session ID
     * @param req Express request object
     */
    static async createUserSession(userId: string, sessionId: string, req: Partial<Request>): Promise<SessionUser> {
        // Create time frame to find schedule data for. 
        const now = new Date();
        const startDate = now;
        const endDate = new Date(now.setDate(now.getDate() + WEEKS_1_DAYS));
        // Query for user data
        const { reminderList, resourceList, ...focusModeSelection } = focusModeSelect(startDate, endDate);
        const userData = await prismaInstance.user.findUnique({
            where: { id: userId },
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
                focusModes: { select: focusModeSelection },
                sessions: {
                    where: { id: sessionId },
                    select: {
                        id: true,
                        last_refresh_at: true,
                    },
                },
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
        if (!userData || userData.sessions.length === 0) {
            throw new CustomError("0510", "NotFound", { userData });
        }
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
        const currentActiveFocusMode = SessionService.getUser(req.session as SessionData)?.activeFocusMode;
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
        const result: SessionUser = {
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
            id: userId,
            languages,
            membershipsCount: userData._count?.memberships ?? 0,
            name: userData.name,
            notesCount: userData._count?.notes ?? 0,
            profileImage: userData.profileImage,
            projectsCount: userData._count?.projects ?? 0,
            questionsAskedCount: userData._count?.questionsAsked ?? 0,
            routinesCount: userData._count?.routines ?? 0,
            session: {
                __typename: "SessionUserSession" as const,
                id: sessionId,
                lastRefreshAt: userData.sessions[0].last_refresh_at,
            },
            standardsCount: userData._count?.standards ?? 0,
            theme: userData.theme,
            updated_at: userData.updated_at,
        };
        return result;
    }

    /**
     * Creates session object from user and existing session data
     * @param userId User ID
     * @param sessionId Session ID
     * @param req Express request object (with current session data)
     * @returns Updated session object, with user data added to the START of the users array
     */
    static async createSession(userId: string, sessionId: string, req: Partial<Request>): Promise<Session> {
        const sessionUser = await SessionService.createUserSession(userId, sessionId, req);
        return {
            __typename: "Session" as const,
            isLoggedIn: true,
            // Make sure users are unique by id
            users: [sessionUser, ...(req.session?.users ?? []).filter((u: SessionUserToken) => u.id !== sessionUser.id).map(SessionService.sessionUserTokenToUser)],
        };
    }

    /**
     * Converts SessionUserToken object o SessionUser object, 
     * by adding dummy fields. Useful for avoiding type errors
     * @param user SessionUserToken object
     * @returns SessionUser object
     */
    static sessionUserTokenToUser(user: SessionUserToken): SessionUser {
        return {
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
            session: {
                __typename: "SessionUserSession" as const,
                id: user.session?.id ?? "",
                lastRefreshAt: user.session?.lastRefreshAt ?? new Date().toISOString(),
            }
        } as const;
    }
}