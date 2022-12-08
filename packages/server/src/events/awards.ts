/**
 * Given a trigger and the trigger's data, determines if the user should receive an
 * award
 */

import { Notify } from "../notify";
import { PrismaType } from "../types";
import i18next, { TFuncKey } from 'i18next';

/**
 * All award categories. An award must belong to only one category, 
 * and a category must have at least one award.
 */
export type AwardCategory = 'AccountAnniversary' |
    'AccountNew' |
    'ApiCreate' |
    'CommentCreate' |
    'IssueCreate' |
    'NoteCreate' |
    'ObjectStar' |
    'ObjectVote' |
    'OrganizationCreate' |
    'OrganizationJoin' |
    'PostCreate' |
    'ProjectCreate' |
    'PullRequestCreate' |
    'PullRequestComplete' |
    'QuestionAnswer' |
    'QuestionCreate' |
    'QuizPass' |
    'ReportEnd' |
    'ReportContribute' |
    'Reputation' |
    'RunRoutine' |
    'RunProject' |
    'RoutineCreate' |
    'SmartContractCreate' |
    'StandardCreate' |
    'Streak' |
    'UserInvite';

type AwardKey = TFuncKey<'award', undefined>

/**
 * Given an ordered list of numbers, returns the closest lower number in the list
 * @param num The number to find the closest lower number for
 * @param list The list of numbers to search, from lowest to highest
 * @returns The closest lower number in the list, or null if there is none
 */
function closestLower(num: number, list: number[]): number | null {
    for (let i = 0; i < list.length; i++) {
        if (list[i] > num)
            return list[i - 1] || null;
    }
    return null;
}

/**
 * Maps award categories to their tiers, if applicable. Special cases are handled
 * in the shouldAward function.
 */
export const awardVariants: { [key in Exclude<AwardCategory, 'AccountAnniversary' | 'AccountNew'>]: number[] } = {
    Streak: [7, 30, 100, 200, 365, 500, 750, 1000],
    QuizPass: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    Reputation: [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
    ObjectStar: [1, 100, 500],
    ObjectVote: [1, 100, 1000, 10000],
    PullRequestCreate: [1, 5, 10, 25, 50, 100, 250, 500],
    PullRequestComplete: [1, 5, 10, 25, 50, 100, 250, 500],
    ApiCreate: [1, 5, 10, 25, 50],
    CommentCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    IssueCreate: [1, 5, 10, 25, 50, 100, 250],
    NoteCreate: [1, 5, 10, 25, 50, 100],
    OrganizationCreate: [1, 2, 5, 10],
    OrganizationJoin: [1, 5, 10, 25],
    PostCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    ProjectCreate: [1, 5, 10, 25, 50, 100],
    QuestionAnswer: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    QuestionCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    ReportEnd: [1, 5, 10, 25, 50, 100],
    ReportContribute: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    RunRoutine: [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
    RunProject: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    RoutineCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    SmartContractCreate: [1, 5, 10, 25],
    StandardCreate: [1, 5, 10, 25, 50],
    UserInvite: [1, 5, 10, 25, 50, 100],
};

/**
 * Determines award tier name from list of names and count
 * @param list The list of names to choose from, paired with 
 * the minimum count required to receive that name. Counts 
 * should be in ascending order.
 * @param count The current count for the award
 * @returns Highest count and name that's applicable
 */
const awardTier = (list: [number, AwardKey][], count: number): AwardKey | null => {
    for (let i = 0; i < list.length; i++) {
        const [min] = list[i];
        if (count < min)
            return list[i - 1][1] || null;
    }
    return null;
}

/**
 * Maps award category/level to the award's name and description. Names should be interesting and unique.
 */
const awardNames: { [key in AwardCategory]: (count: number) => {
    name: AwardKey | null,
    nameVariables?: { count: number },
    body: AwardKey | null,
    bodyVariables?: { count: number },
} } = {
    AccountAnniversary: (years: number) => ({
        name: 'AccountAnniversaryTitle',
        nameVariables: { count: years },
        body: 'AccountAnniversaryBody',
        bodyVariables: { count: years },
    }),
    AccountNew: () => ({ name: 'AccountNewTitle', body: 'AccountNewBody' }),
    Streak: (days: number) => {
        const tit = <C extends number>(count: C) => `${'StreakDays'}${count}Title` as const;
        const name = awardTier([[7, tit(7)], [30, tit(30)], [100, tit(100)], [200, tit(200)], [365, tit(365)], [500, tit(500)], [750, tit(750)], [1000, tit(1000)]], days);
        if (!name) return { name: null, body: null };
        return { name, body: 'StreakDaysBody', bodyVariables: { count: days } };
    },
    QuizPass: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuizPass'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'QuizPassBody', bodyVariables: { count } };
    },
    Reputation: (count: number) => {
        // [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${'ReputationPoints'}${count}Title` as const
        const name = awardTier([[10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ReputationPointsBody', bodyVariables: { count } };
    },
    ObjectStar: (count: number) => {
        // [1, 100, 500]
        const tit = <C extends number>(count: C) => `${'ObjectStar'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [100, tit(100)], [500, tit(500)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ObjectStarBody', bodyVariables: { count } };
    },
    ObjectVote: (count: number) => {
        // [1, 100, 1000, 10000]
        const tit = <C extends number>(count: C) => `${'ObjectVote'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [100, tit(100)], [1000, tit(1000)], [10000, tit(10000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ObjectVoteBody', bodyVariables: { count } };
    },
    PullRequestCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${'PullRequestCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'PullRequestCreateBody', bodyVariables: { count } };
    },
    PullRequestComplete: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${'PullRequestComplete'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'PullRequestCompleteBody', bodyVariables: { count } };
    },
    ApiCreate: (count: number) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${'ApiCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ApiCreateBody', bodyVariables: { count } };
    },
    CommentCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'CommentCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'CommentCreateBody', bodyVariables: { count } };
    },
    IssueCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250]
        const tit = <C extends number>(count: C) => `${'IssueCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'IssueCreateBody', bodyVariables: { count } };
    },
    NoteCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'NoteCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'NoteCreateBody', bodyVariables: { count } };
    },
    OrganizationCreate: (count: number) => {
        // [1, 2, 5, 10]
        const tit = <C extends number>(count: C) => `${'OrganizationCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [2, tit(2)], [5, tit(5)], [10, tit(10)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'OrganizationCreateBody', bodyVariables: { count } };
    },
    OrganizationJoin: (count: number) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${'OrganizationJoin'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'OrganizationJoinBody', bodyVariables: { count } };
    },
    PostCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'PostCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'PostCreateBody', bodyVariables: { count } };
    },
    ProjectCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'PostCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'PostCreateBody', bodyVariables: { count } };
    },
    QuestionAnswer: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuestionAnswer'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'QuestionAnswerBody', bodyVariables: { count } };
    },
    QuestionCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuestionCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'QuestionCreateBody', bodyVariables: { count } };
    },
    ReportEnd: (count: number) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'ReportEnd'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ReportEndBody', bodyVariables: { count } };
    },
    ReportContribute: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'ReportContribute'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'ReportContributeBody', bodyVariables: { count } };
    },
    RunRoutine: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${'RunRoutine'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'RunRoutineBody', bodyVariables: { count } };
    },
    RunProject: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'RunProject'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'RunProjectBody', bodyVariables: { count } };
    },
    RoutineCreate: (count: number) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000
        const tit = <C extends number>(count: C) => `${'RoutineCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'RoutineCreateBody', bodyVariables: { count } };
    },
    SmartContractCreate: (count: number) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${'SmartContractCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'SmartContractCreateBody', bodyVariables: { count } };
    },
    StandardCreate: (count: number) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${'StandardCreate'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'StandardCreateBody', bodyVariables: { count } };
    },
    UserInvite: (count: number) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'UserInvite'}${count}Title` as const
        const name = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count);
        if (!name) return { name: null, body: null };
        return { name, body: 'UserInviteBody', bodyVariables: { count } };
    },
}

/**
 * Checks if a user should receive an award
 * @param awardCategory The award category
 * @param previousCount The previous count of the award category
 * @param currentCount The current count of the award category
 * @returns True if the user should receive the award
 */
const shouldAward = (awardCategory: AwardCategory, previousCount: number, currentCount: number): boolean => {
    // Anniversary and new accounts are special cases
    if (awardCategory === 'AccountAnniversary') return currentCount > previousCount;
    if (awardCategory === 'AccountNew') return false;
    // Get tiers
    const tiers = awardVariants[awardCategory];
    if (!tiers) return false;
    // Check which tier previous count and new count are in
    const previous = closestLower(previousCount, tiers);
    const current = closestLower(currentCount, tiers);
    // Only award if moving to a new tier
    return current !== null && current > (previous ?? 0);
}

/**
 * Handles tracking awards for a user. If a new award is earned, a notification
 * can be sent to the user (push or email)
 */
export const Award = (prisma: PrismaType, userId: string, languages: string[]) => ({
    /**
     * Upserts an award into the database. If the award progress reaches a new goal,
     * the user is notified
     * @param category The category of the award
     * @param newProgress The new progress of the award
     * @param languages Preferred languages for the award name and body
     */
    update: async (category: AwardCategory, newProgress: number) => {
        // Upsert the award into the database, with progress incremented
        // by the new progress
        const award = await prisma.award.upsert({
            where: { userId_category: { userId, category } },
            update: { progress: { increment: newProgress } },
            create: { userId, category, progress: newProgress },
        });
        // Check if user should receive a new award (i.e. the progress has put them
        // into a new award tier)
        const isNewTier = shouldAward(category, award.progress - newProgress, award.progress);
        if (isNewTier) {
            // Get translated award name and body
            const lng = languages.length > 0 ? languages[0] : 'en';
            const { name, nameVariables, body, bodyVariables } = awardNames[category](award.progress);
            const transTitle = name ? i18next.t(`award:${name}`, { lng, ...(nameVariables ?? {}) }) : null;
            const transBody = body ? i18next.t(`award:${body}`, { lng, ...(bodyVariables ?? {}) }) : null;
            // Send a notification to the user
            if (transTitle && transBody) {
                await Notify(prisma, languages).pushAward(transTitle, transBody).toUser(userId);
            }
            // Set "timeCurrentTierCompleted" to the current time
            await prisma.award.update({
                where: { userId_category: { userId, category } },
                data: { timeCurrentTierCompleted: new Date() },
            });
        }
        return award;
    },
})