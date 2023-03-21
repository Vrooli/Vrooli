import { Session, SessionUser } from '@shared/consts';
import { Request } from "express";
import { CustomError } from "../events";
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
            focusModes: {
                // Schedules have an event start/end and recurring start/end.
                // Select all schedules which are occuring now or in the near future (7 days)
                where: {}, //TODO
                select: {
                    id: true,
                } //TODO
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
        __typename: 'SessionUser' as const,
        focusModes: [] as any[], //TODO
        handle: userData.handle ?? undefined,
        hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
        id: user.id,
        languages,
        name: userData.name,
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
        __typename: 'Session' as const,
        isLoggedIn: true,
        // Make sure users are unique by id
        users: [sessionUser, ...(req.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
    }
}