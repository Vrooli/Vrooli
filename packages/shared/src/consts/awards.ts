/* eslint-disable no-magic-numbers */
import { AwardCategory } from "../api/types.js";
import { type TranslationKeyAward } from "../types.js";

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
    SmartContractCreate: [1, 5, 10, 25],
    CommentCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    IssueCreate: [1, 5, 10, 25, 50, 100, 250],
    NoteCreate: [1, 5, 10, 25, 50, 100],
    PostCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    ProjectCreate: [1, 5, 10, 25, 50, 100],
    QuestionAnswer: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    QuestionCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    ReportEnd: [1, 5, 10, 25, 50, 100],
    ReportContribute: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    RunRoutine: [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
    RunProject: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    RoutineCreate: [1, 5, 10, 25, 50, 100, 250, 500, 1000],
    StandardCreate: [1, 5, 10, 25, 50],
    OrganizationCreate: [1, 2, 5, 10],
    OrganizationJoin: [1, 5, 10, 25],
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
function awardTier(
    list: [number, TranslationKeyAward][],
    count: number,
    findNext = false,
): [number, TranslationKeyAward | null] {
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
}

function title<Prefix extends string, Count extends number>(prefix: Prefix, count: Count): `${Prefix}${Count}_Title` {
    return `${prefix}${count}_Title` as const;
}

/**
 * Maps award category/level to the award's name and description. Names should be interesting and unique.
 */
export const awardNames: { [key in AwardCategory]: (count: number, findNext?: boolean) => {
    name: TranslationKeyAward | null,
    nameVariables?: { count: number },
    body: TranslationKeyAward | null,
    bodyVariables?: { count: number },
    level: number,
} } = {
    AccountAnniversary: (years, findNext = false) => ({
        name: "AccountAnniversary_Title",
        nameVariables: { count: findNext ? years + 1 : years },
        body: "AccountAnniversary_Body",
        bodyVariables: { count: findNext ? years + 1 : years },
        level: findNext ? years + 1 : years,
    }),
    AccountNew: () => ({
        name: "AccountNew_Title",
        body: "AccountNew_Body",
        level: 0,
    }),
    ApiCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const p = "ApiCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ApiCreate_Body", bodyVariables: { count: level }, level };
    },
    SmartContractCreate: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const p = "SmartContractCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "SmartContractCreate_Body", bodyVariables: { count: level }, level };
    },
    CommentCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "CommentCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "CommentCreate_Body", bodyVariables: { count: level }, level };
    },
    IssueCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250]
        const p = "IssueCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "IssueCreate_Body", bodyVariables: { count: level }, level };
    },
    NoteCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const p = "NoteCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "NoteCreate_Body", bodyVariables: { count: level }, level };
    },
    ObjectBookmark: (count, findNext = false) => {
        // [1, 100, 500]
        const p = "ObjectBookmark";
        const [level, name] = awardTier([[1, title(p, 1)], [100, title(p, 100)], [500, title(p, 500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ObjectBookmark_Body", bodyVariables: { count: level }, level };
    },
    ObjectReact: (count, findNext = false) => {
        // [1, 100, 1000, 10000]
        const p = "ObjectReact";
        const [level, name] = awardTier([[1, title(p, 1)], [100, title(p, 100)], [1000, title(p, 1000)], [10000, title(p, 10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ObjectReact_Body", bodyVariables: { count: level }, level };
    },
    PostCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "PostCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PostCreate_Body", bodyVariables: { count: level }, level };
    },
    ProjectCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const p = "ProjectCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ProjectCreate_Body", bodyVariables: { count: level }, level };
    },
    PullRequestCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const p = "PullRequestCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestCreate_Body", bodyVariables: { count: level }, level };
    },
    PullRequestComplete: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500]
        const p = "PullRequestComplete";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestComplete_Body", bodyVariables: { count: level }, level };
    },
    QuestionAnswer: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "QuestionAnswer";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuestionAnswer_Body", bodyVariables: { count: level }, level };
    },
    QuestionCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "QuestionCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuestionCreate_Body", bodyVariables: { count: level }, level };
    },
    QuizPass: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "QuizPass";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "QuizPass_Body", bodyVariables: { count: level }, level };
    },
    ReportEnd: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const p = "ReportEnd";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReportEnd_Body", bodyVariables: { count: level }, level };
    },
    ReportContribute: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "ReportContribute";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReportContribute_Body", bodyVariables: { count: level }, level };
    },
    Reputation: (count, findNext = false) => {
        // [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const p = "ReputationPoints";
        const [level, name] = awardTier([[10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)], [2500, title(p, 2500)], [10000, title(p, 10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "ReputationPoints_Body", bodyVariables: { count: level }, level };
    },
    RunRoutine: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
        const p = "RunRoutine";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)], [2500, title(p, 2500)], [10000, title(p, 10000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RunRoutine_Body", bodyVariables: { count: level }, level };
    },
    RunProject: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000]
        const p = "RunProject";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RunProject_Body", bodyVariables: { count: level }, level };
    },
    RoutineCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100, 250, 500, 1000
        const p = "RoutineCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)], [250, title(p, 250)], [500, title(p, 500)], [1000, title(p, 1000)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "RoutineCreate_Body", bodyVariables: { count: level }, level };
    },
    StandardCreate: (count, findNext = false) => {
        // [1, 5, 10, 25, 50]
        const p = "StandardCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "StandardCreate_Body", bodyVariables: { count: level }, level };
    },
    Streak: (days, findNext = false) => {
        // [7, 30, 100, 200, 365, 500, 750, 1000]
        const p = "StreakDays";
        const [level, name] = awardTier([[7, title(p, 7)], [30, title(p, 30)], [100, title(p, 100)], [200, title(p, 200)], [365, title(p, 365)], [500, title(p, 500)], [750, title(p, 750)], [1000, title(p, 1000)]], days, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "StreakDays_Body", bodyVariables: { count: days }, level };
    },
    OrganizationCreate: (count, findNext = false) => {
        // [1, 2, 5, 10]
        const p = "OrganizationCreate";
        const [level, name] = awardTier([[1, title(p, 1)], [2, title(p, 2)], [5, title(p, 5)], [10, title(p, 10)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationCreate_Body", bodyVariables: { count: level }, level };
    },
    OrganizationJoin: (count, findNext = false) => {
        // [1, 5, 10, 25]
        const p = "OrganizationJoin";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationJoin_Body", bodyVariables: { count: level }, level };
    },
    UserInvite: (count, findNext = false) => {
        // [1, 5, 10, 25, 50, 100]
        const p = "UserInvite";
        const [level, name] = awardTier([[1, title(p, 1)], [5, title(p, 5)], [10, title(p, 10)], [25, title(p, 25)], [50, title(p, 50)], [100, title(p, 100)]], count, findNext);
        if (!name) return { name: null, body: null, level: 0 };
        return { name, body: "UserInvite_Body", bodyVariables: { count: level }, level };
    },
};
