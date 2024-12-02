import { FocusModeStopCondition, Session, SessionUser, uuidValidate } from "@local/shared";
import { Request } from "express";
import { CustomError } from "../events/error";
import { scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events/schedule";
import { SessionData } from "../types";
import { UserDataForPasswordAuth } from "./email";

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
    static getUser(session: Pick<SessionData, "users">): SessionUser | null {
        if (!session || !Array.isArray(session?.users) || session.users.length === 0) return null;
        const user = session.users[0];
        return user !== undefined && typeof user.id === "string" && uuidValidate(user.id) ? user : null;
    }

    /**
     * Creates SessionUser object from user.
     * @param userData User data
     * @param session Current session data
     */
    static async createUserSession(userData: Omit<UserDataForPasswordAuth, "auths" | "emails" | "sessions">, sessionData: UserDataForPasswordAuth["sessions"][0]): Promise<SessionUser> {
        if (!userData || !sessionData) {
            throw new CustomError("0510", "NotFound", { userData });
        }
        // Return shaped SessionUser object
        const result: SessionUser = {
            __typename: "SessionUser" as const,
            id: userData.id,
            activeFocusMode: userData.activeFocusMode ? {
                __typename: "ActiveFocusMode" as const,
                focusMode: {
                    __typename: "ActiveFocusModeFocusMode" as const,
                    id: userData.activeFocusMode.focusMode.id,
                    reminderListId: userData.activeFocusMode.focusMode.reminderListId,
                },
                stopCondition: userData.activeFocusMode.stopCondition as FocusModeStopCondition,
                stopTime: userData.activeFocusMode.stopTime,
            } : null,
            apisCount: userData._count?.apis ?? 0,
            codesCount: userData._count?.codes ?? 0,
            credits: (userData.premium?.credits ?? BigInt(0)) + "", // Convert to string because BigInt can't be serialized
            handle: userData.handle ?? undefined,
            hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
            languages: userData.languages.map(l => l.language),
            membershipsCount: userData._count?.memberships ?? 0,
            name: userData.name,
            notesCount: userData._count?.notes ?? 0,
            profileImage: userData.profileImage,
            projectsCount: userData._count?.projects ?? 0,
            questionsAskedCount: userData._count?.questionsAsked ?? 0,
            routinesCount: userData._count?.routines ?? 0,
            session: {
                __typename: "SessionUserSession" as const,
                id: sessionData.id,
                lastRefreshAt: sessionData.last_refresh_at,
            },
            standardsCount: userData._count?.standards ?? 0,
            theme: userData.theme,
            updated_at: userData.updated_at,
        };
        return result;
    }

    static createBaseSession(req: Request): Session {
        return {
            __typename: "Session" as const,
            isLoggedIn: req.session?.isLoggedIn ?? false,
            timeZone: req.session?.timeZone,
            users: req.session?.users ?? [],
        };
    }

    /**
     * Creates session object from user and existing session data
     * @param userData User data
     * @param session Current session data
     * @param req Express request object (with current session data)
     * @returns Updated session object, with user data added to the START of the users array
     */
    static async createSession(userData: UserDataForPasswordAuth, sessionData: UserDataForPasswordAuth["sessions"][0], req: Request): Promise<Session> {
        const sessionUser = await SessionService.createUserSession(userData, sessionData);
        return {
            ...this.createBaseSession(req),
            isLoggedIn: true,
            // Make sure users are unique by id
            users: [sessionUser, ...(req.session?.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
        };
    }
}
