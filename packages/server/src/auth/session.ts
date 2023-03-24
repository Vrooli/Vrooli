import { Session, SessionUser } from '@shared/consts';
import { getActiveFocusMode } from '@shared/utils';
import { Request } from "express";
import { CustomError, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe } from "../events";
import { PrismaType } from "../types";
import { getUser } from './request';

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
    console.log('tosessionuser a', user.id)
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
            focusModes: {
                select: {
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
                }
            }
        }
    })
    console.log('tosessionuser b', JSON.stringify(userData), '\n\n');
    // Find active focus mode
    const activeFocusMode = getActiveFocusMode(getUser(req)?.activeFocusMode, (userData.focusModes as any) ?? []);
    console.log('tosessionuser c', JSON.stringify(activeFocusMode), '\n\n')
    // Calculate langugages, by combining user's languages with languages 
    // in request. Make sure to remove duplicates
    let languages: string[] = userData.languages.map((l) => l.language).filter(Boolean) as string[];
    if (req.languages) languages.push(...req.languages);
    languages = [...new Set(languages)];
    console.log('tosessionuser d', JSON.stringify(languages), '\n\n')
    // Return shaped SessionUser object
    const result = {
        __typename: 'SessionUser' as const,
        activeFocusMode,
        focusModes: userData.focusModes as any[],
        handle: userData.handle ?? undefined,
        hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
        id: user.id,
        languages,
        name: userData.name,
        theme: userData.theme,
    }
    console.log('tosessionuser e', JSON.stringify(result), '\n\n')
    return result;
}

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
        users: [sessionUser, ...(req.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
    }
}