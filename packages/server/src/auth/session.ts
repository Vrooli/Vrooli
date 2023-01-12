import { Request } from "express";
import { CustomError } from "../events";
import { GqlModelType, Session, SessionUser } from '@shared/consts';
import { PrismaType } from "../types";

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
    // Get UNIX timestamp for 7 days from now
    const in7Days = new Date().getTime() + 7 * 24 * 60 * 60 * 1000;
    // Update user's lastSessionVerified
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
            schedules: {
                // Schedules have an event start/end and recurring start/end.
                // Select all schedules which are occuring now or in the near future (7 days)
                where: {
                    OR: [
                        // Event start is in the past (or in the near future), and event end is in the future
                        { eventStart: { lte: new Date(in7Days).toISOString() }, eventEnd: { gte: new Date().toISOString() } },
                        // Recurring start is in the past (or in the near future), and recurring end is in the future
                        { recurring: true, recurrStart: { lte: new Date(in7Days).toISOString() }, recurrEnd: { gte: new Date().toISOString() } },
                    ]
                },
                select: {
                    id: true,
                    name: true,
                    description: true,
                    timeZone: true,
                    eventStart: true,
                    eventEnd: true,
                    recurring: true,
                    recurrStart: true,
                    recurrEnd: true,
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
                    resourceList: {
                        select: {
                            id: true,
                            // TODO
                        }
                    },
                    reminderList: { 
                        select: {
                            id: true,
                            // TODO
                        }
                    }
                }
            }
        }
    })
    // Calculate langugages, by combining user's languages with languages 
    // in request. Make sure to remove duplicates
    let languages: string[] = userData.languages.map((l) => l.language).filter(Boolean) as string[];
    if (req.languages) languages.push(...req.languages);
    languages = [...new Set(languages)];
    // Return shaped SessionUser object
    return {
        handle: userData.handle ?? undefined,
        hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
        id: user.id,
        languages,
        name: userData.name,
        schedules: userData.schedules as any[], //TODO
        theme: userData.theme,
    }
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
        type: GqlModelType.Session,
        isLoggedIn: true,
        // Make sure users are unique by id
        users: [sessionUser, ...(req.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
    }
}