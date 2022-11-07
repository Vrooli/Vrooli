/**
 * Given a trigger and the trigger's data, determines if the user should receive an
 * award
 */

import { PrismaType } from "../types";
import { ActionTrigger } from "./trigger"

/**
 * All award categories. An award must belong to only one category, 
 * and a category must have at least one award.
 */
export enum AwardCategory {
    AccountAnniversary = 'AccountAnniversary',
    AccountNew = 'AccountNew',
    ApiCreate = 'ApiCreate',
    CommentCreate = 'CommentCreate',
    IssueCreate = 'IssueCreate',
    NoteCreate = 'NoteCreate',
    ObjectStar = 'ObjectStar',
    ObjectVote = 'ObjectVote',
    OrganizationCreate = 'OrganizationCreate',
    OrgnanizationJoin = 'OrgnanizationJoin',
    PostCreate = 'PostCreate',
    ProjectCreate = 'ProjectCreate',
    PullRequestCreate = 'PullRequestCreate',
    PullRequestComplete = 'PullRequestComplete',
    QuestionAnswer = 'QuestionAnswer',
    QuestionCreate = 'QuestionCreate',
    QuizComplete = 'QuizComplete',
    ReportEnd = 'ReportEnd',
    ReportContribute = 'ReportContribute',
    Reputation = 'Reputation',
    RoutineComplete = 'RoutineComplete',
    RoutineCompleteLearning = 'RoutineCompleteLearning',
    RoutineCompleteOnChristmas = 'RoutineCompleteOnChristmas', // Easter egg
    RoutineCreate = 'RoutineCreate',
    SmartContractCreate = 'SmartContractCreate',
    StandardCreate = 'StandardCreate',
    Streak = 'Streak',
    UserInvite = 'UserInvite',
}

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

// Determines variant for awards. Example: 7-day streak, 100th routine completed, etc.
// If an award has a variant, returns the closest lower variant (i.e. the highest variant that's applicable)
export const awardVariants: { [key in AwardCategory]?: (count: number) => number | null } = {
    [AwardCategory.AccountAnniversary]: (years: number) => years,
    // No variants for AccountNew
    [AwardCategory.AccountNew]: () => null,
    [AwardCategory.Streak]: (days: number) => closestLower(days, [7, 30, 100, 200, 365, 500, 750, 1000]),
    [AwardCategory.QuizComplete]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.Reputation]: (count: number) => closestLower(count, [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]),
    [AwardCategory.ObjectStar]: (count: number) => closestLower(count, [100, 500]),
    [AwardCategory.ObjectVote]: (count: number) => closestLower(count, [100, 1000, 10000]),
    [AwardCategory.PullRequestCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.PullRequestComplete]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.ApiCreate]: (count: number) => closestLower(count, [5, 10, 25, 50]),
    [AwardCategory.CommentCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.IssueCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250]),
    [AwardCategory.NoteCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100]),
    [AwardCategory.OrganizationCreate]: (count: number) => closestLower(count, [2, 5, 10]),
    [AwardCategory.OrgnanizationJoin]: (count: number) => closestLower(count, [5, 10, 25]),
    [AwardCategory.PostCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.ProjectCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100]),
    [AwardCategory.QuestionAnswer]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.QuestionCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.ReportEnd]: (count: number) => closestLower(count, [5, 10, 25, 50, 100]),
    [AwardCategory.ReportContribute]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.RoutineComplete]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]),
    [AwardCategory.RoutineCompleteLearning]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.RoutineCompleteOnChristmas]: () => null,
    [AwardCategory.RoutineCreate]: (count: number) => closestLower(count, [5, 10, 25, 50, 100, 250, 500, 1000]),
    [AwardCategory.SmartContractCreate]: (count: number) => closestLower(count, [5, 10, 25]),
    [AwardCategory.StandardCreate]: (count: number) => closestLower(count, [5, 10, 25, 50]),
    [AwardCategory.UserInvite]: (count: number) => closestLower(count, [5, 10, 25, 50, 100]),
};

// TODO functions to check if user should receive award

/**
 * Maps trigger events to award functions which determine if the user should receive
 * an award
 */
// export const TriggersToAwards: { [key in ActionTrigger]?: [] } = {
//     [ActionTrigger.Create]: []
// }

/**
 * Handles tracking awards for a user. If a new award is earned, a notification
 * can be sent to the user (push or email)
 */
export const Award = (prisma: PrismaType, userId: string) => ({
    /**
     * Upserts an award into the database. If the award progress reaches a new goal,
     * the user is notified
     * @param category The category of the award
     * @param progress The progress of the award
     */
    update: async (category: AwardCategory, progress: number): any => {
        // const award = await prisma.award.upsert({
        //     where: {
        //         userId_category: {
        //             userId,
        //             category,
        //         },
        //     },
        //     update: {
        //         progress,
        //     },
        //     create: {
        //         userId,
        //         category,
        //         progress,
        //     },
        // });
        // return award;
    },
})