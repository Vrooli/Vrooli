import { AwardKey } from "@shared/translations";
import { AwardCategory } from "./graphqlTypes";

/**
 * Maps award categories to their tiers, if applicable. Special cases are handled
 * in the shouldAward function.
 */
export const awardVariants: { [key in Exclude<`${AwardCategory}`, 'AccountAnniversary' | 'AccountNew'>]: number[] } = {
    Streak: [7, 30, 100, 200, 365, 500, 750, 1000],
    QuizPass: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    Reputation: [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
    ObjectBookmark: [1, 100, 500],
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
 * @param findNext If true, will return the next tier name instead of the current one
 * @returns Highest count and name that's applicable
 */
const awardTier = (
    list: [number, AwardKey][],
    count: number,
    findNext: boolean = false,
): [number, AwardKey] | null => {
    for (let i = 0; i < list.length; i++) {
        const [min] = list[i];
        if (count < min) {
            const returnIndex = findNext ? i : Math.min(i + 1, list.length - 1);
            return list[returnIndex] || null;
        }
            
    }
    return null;
}

/**
 * Maps award category/level to the award's name and description. Names should be interesting and unique.
 */
export const awardNames: { [key in AwardCategory]: (count: number, findNext?: boolean) => {
    name: AwardKey | null,
    nameVariables?: { count: number },
    body: AwardKey | null,
    bodyVariables?: { count: number },
    level: number,
} } = {
    AccountAnniversary: (years, findNext = false) => ({
        name: 'AccountAnniversaryTitle',
        nameVariables: { count: findNext ? years + 1 : years },
        body: 'AccountAnniversaryBody',
        bodyVariables: { count: findNext ? years + 1 : years },
        level: findNext ? years + 1 : years,
    }),
    AccountNew: () => ({ name: 'AccountNewTitle', body: 'AccountNewBody', level: 0 }),
    Streak: (days, findNext = false) => {
        const tit = <C extends number>(count: C) => `${'StreakDays'}${count}Title` as const;
        const tier = awardTier([[7, tit(7)], [30, tit(30)], [100, tit(100)], [200, tit(200)], [365, tit(365)], [500, tit(500)], [750, tit(750)], [1000, tit(1000)]], days, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'StreakDaysBody', bodyVariables: { count: days }, level: tier[0] };
    },
    QuizPass: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuizPass'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'QuizPassBody', bodyVariables: { count }, level: tier[0] };
    },
    Reputation: (count, findNext = false) => {
        // [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${'ReputationPoints'}${count}Title` as const
        const tier = awardTier([[10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ReputationPointsBody', bodyVariables: { count }, level: tier[0] };
    },
    ObjectBookmark: (count, findNext = false) => {
        // [1, 100, 500]
        const tit = <C extends number>(count: C) => `${'ObjectBookmark'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [100, tit(100)], [500, tit(500)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ObjectBookmarkBody', bodyVariables: { count }, level: tier[0] };
    },
    ObjectVote: (count, findNext = false) => {
        // [1, 100, 1000, 10000]
        const tit = <C extends number>(count: C) => `${'ObjectVote'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [100, tit(100)], [1000, tit(1000)], [10000, tit(10000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ObjectVoteBody', bodyVariables: { count }, level: tier[0] };
    },
    PullRequestCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${'PullRequestCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'PullRequestCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    PullRequestComplete: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${'PullRequestComplete'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'PullRequestCompleteBody', bodyVariables: { count }, level: tier[0] };
    },
    ApiCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${'ApiCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ApiCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    CommentCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'CommentCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'CommentCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    IssueCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250]
        const tit = <C extends number>(count: C) => `${'IssueCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'IssueCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    NoteCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'NoteCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'NoteCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    OrganizationCreate: (count, findNext = false) => {
        // [1, 2, 5, 10]
        const tit = <C extends number>(count: C) => `${'OrganizationCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [2, tit(2)], [5, tit(5)], [10, tit(10)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'OrganizationCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    OrganizationJoin: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${'OrganizationJoin'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'OrganizationJoinBody', bodyVariables: { count }, level: tier[0] };
    },
    PostCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'PostCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'PostCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    ProjectCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'PostCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'PostCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    QuestionAnswer: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuestionAnswer'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'QuestionAnswerBody', bodyVariables: { count }, level: tier[0] };
    },
    QuestionCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'QuestionCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'QuestionCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    ReportEnd: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'ReportEnd'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ReportEndBody', bodyVariables: { count }, level: tier[0] };
    },
    ReportContribute: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'ReportContribute'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'ReportContributeBody', bodyVariables: { count }, level: tier[0] };
    },
    RunRoutine: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${'RunRoutine'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'RunRoutineBody', bodyVariables: { count }, level: tier[0] };
    },
    RunProject: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${'RunProject'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'RunProjectBody', bodyVariables: { count }, level: tier[0] };
    },
    RoutineCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000
        const tit = <C extends number>(count: C) => `${'RoutineCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'RoutineCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    SmartContractCreate: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${'SmartContractCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'SmartContractCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    StandardCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${'StandardCreate'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'StandardCreateBody', bodyVariables: { count }, level: tier[0] };
    },
    UserInvite: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${'UserInvite'}${count}Title` as const
        const tier = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!tier) return { name: null, body: null, level: 0 };
        return { name: tier[1], body: 'UserInviteBody', bodyVariables: { count }, level: tier[0] };
    },
}