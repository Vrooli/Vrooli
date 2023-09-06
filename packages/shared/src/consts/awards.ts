import { AwardKey } from "@local/shared";
import { AwardCategory } from "../api/generated/graphqlTypes";

/**
 * Maps award categories to their tiers, if applicable. Special cases are handled
 * in the shouldAward function.
 */
export const awardVariants: { [key in Exclude<`${AwardCategory}`, "AccountAnniversary" | "AccountNew">]: number[] } = {
    Streak: [7, 30, 100, 200, 365, 500, 750, 1000],
    QuizPass: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    Reputation: [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
    ObjectBookmark: [1, 100, 500],
    ObjectReact: [1, 100, 1000, 10000],
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
    findNext = false,
): [number, AwardKey | null] => {
    for (let i = 0; i < list.length; i++) {
        const item = list[i];
        if (!item) continue;
        const [min] = item;
        if (count < min) {
            const returnIndex = findNext ? Math.min(i + 1, list.length - 1) : i;
            return list[returnIndex] ?? [0, null];
        }

    }
    // If not found and findNext is true, return the first tier
    const foundResult = list.length > 0 ? list[list.length - 1] : undefined;
    return foundResult ?? [0, null];
};

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
        name: "AccountAnniversaryTitle",
        nameVariables: { count: findNext ? years + 1 : years },
        body: "AccountAnniversaryBody",
        bodyVariables: { count: findNext ? years + 1 : years },
        level: findNext ? years + 1 : years,
    }),
    AccountNew: () => ({ name: "AccountNewTitle", body: "AccountNewBody", level: 0 }),
    ApiCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${"ApiCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ApiCreateBody", bodyVariables: { count: level }, level };
    },
    CommentCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"CommentCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "CommentCreateBody", bodyVariables: { count: level }, level };
    },
    IssueCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250]
        const tit = <C extends number>(count: C) => `${"IssueCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "IssueCreateBody", bodyVariables: { count: level }, level };
    },
    NoteCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${"NoteCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "NoteCreateBody", bodyVariables: { count: level }, level };
    },
    ObjectBookmark: (count, findNext = false) => {
        // [1, 100, 500]
        const tit = <C extends number>(count: C) => `${"ObjectBookmark"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [100, tit(100)], [500, tit(500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ObjectBookmarkBody", bodyVariables: { count: level }, level };
    },
    ObjectReact: (count, findNext = false) => {
        // [1, 100, 1000, 10000]
        const tit = <C extends number>(count: C) => `${"ObjectReact"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [100, tit(100)], [1000, tit(1000)], [10000, tit(10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ObjectReactBody", bodyVariables: { count: level }, level };
    },
    OrganizationCreate: (count, findNext = false) => {
        // [1, 2, 5, 10]
        const tit = <C extends number>(count: C) => `${"OrganizationCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [2, tit(2)], [5, tit(5)], [10, tit(10)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationCreateBody", bodyVariables: { count: level }, level };
    },
    OrganizationJoin: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${"OrganizationJoin"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationJoinBody", bodyVariables: { count: level }, level };
    },
    PostCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"PostCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PostCreateBody", bodyVariables: { count: level }, level };
    },
    ProjectCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${"ProjectCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ProjectCreateBody", bodyVariables: { count: level }, level };
    },
    PullRequestCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${"PullRequestCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestCreateBody", bodyVariables: { count: level }, level };
    },
    PullRequestComplete: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const tit = <C extends number>(count: C) => `${"PullRequestComplete"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestCompleteBody", bodyVariables: { count: level }, level };
    },
    QuestionAnswer: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"QuestionAnswer"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuestionAnswerBody", bodyVariables: { count: level }, level };
    },
    QuestionCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"QuestionCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuestionCreateBody", bodyVariables: { count: level }, level };
    },
    QuizPass: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"QuizPass"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuizPassBody", bodyVariables: { count: level }, level };
    },
    ReportEnd: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${"ReportEnd"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReportEndBody", bodyVariables: { count: level }, level };
    },
    ReportContribute: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"ReportContribute"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReportContributeBody", bodyVariables: { count: level }, level };
    },
    Reputation: (count, findNext = false) => {
        // [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${"ReputationPoints"}${count}Title` as const;
        const [level, name] = awardTier([[10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReputationPointsBody", bodyVariables: { count: level }, level };
    },
    RunRoutine: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const tit = <C extends number>(count: C) => `${"RunRoutine"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RunRoutineBody", bodyVariables: { count: level }, level };
    },
    RunProject: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const tit = <C extends number>(count: C) => `${"RunProject"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RunProjectBody", bodyVariables: { count: level }, level };
    },
    RoutineCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000
        const tit = <C extends number>(count: C) => `${"RoutineCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RoutineCreateBody", bodyVariables: { count: level }, level };
    },
    SmartContractCreate: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const tit = <C extends number>(count: C) => `${"SmartContractCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "SmartContractCreateBody", bodyVariables: { count: level }, level };
    },
    StandardCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const tit = <C extends number>(count: C) => `${"StandardCreate"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "StandardCreateBody", bodyVariables: { count: level }, level };
    },
    Streak: (days, findNext = false) => {
        const tit = <C extends number>(count: C) => `${"StreakDays"}${count}Title` as const;
        const [level, name] = awardTier([[7, tit(7)], [30, tit(30)], [100, tit(100)], [200, tit(200)], [365, tit(365)], [500, tit(500)], [750, tit(750)], [1000, tit(1000)]], days, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "StreakDaysBody", bodyVariables: { count: days }, level };
    },
    UserInvite: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const tit = <C extends number>(count: C) => `${"UserInvite"}${count}Title` as const;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "UserInviteBody", bodyVariables: { count: level }, level };
    },
};
