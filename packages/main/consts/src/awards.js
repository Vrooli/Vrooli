export const awardVariants = {
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
const awardTier = (list, count, findNext = false) => {
    for (let i = 0; i < list.length; i++) {
        const [min] = list[i];
        if (count < min) {
            const returnIndex = findNext ? Math.min(i + 1, list.length - 1) : i;
            return list[returnIndex] || null;
        }
    }
    if (findNext)
        return list.length > 0 ? list[0] : [0, null];
    return [0, null];
};
export const awardNames = {
    AccountAnniversary: (years, findNext = false) => ({
        name: "AccountAnniversaryTitle",
        nameVariables: { count: findNext ? years + 1 : years },
        body: "AccountAnniversaryBody",
        bodyVariables: { count: findNext ? years + 1 : years },
        level: findNext ? years + 1 : years,
    }),
    AccountNew: () => ({ name: "AccountNewTitle", body: "AccountNewBody", level: 0 }),
    ApiCreate: (count, findNext = false) => {
        const tit = (count) => `${"ApiCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ApiCreateBody", bodyVariables: { count: level }, level };
    },
    CommentCreate: (count, findNext = false) => {
        const tit = (count) => `${"CommentCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "CommentCreateBody", bodyVariables: { count: level }, level };
    },
    IssueCreate: (count, findNext = false) => {
        const tit = (count) => `${"IssueCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "IssueCreateBody", bodyVariables: { count: level }, level };
    },
    NoteCreate: (count, findNext = false) => {
        const tit = (count) => `${"NoteCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "NoteCreateBody", bodyVariables: { count: level }, level };
    },
    ObjectBookmark: (count, findNext = false) => {
        const tit = (count) => `${"ObjectBookmark"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [100, tit(100)], [500, tit(500)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ObjectBookmarkBody", bodyVariables: { count: level }, level };
    },
    ObjectReact: (count, findNext = false) => {
        const tit = (count) => `${"ObjectReact"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [100, tit(100)], [1000, tit(1000)], [10000, tit(10000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ObjectReactBody", bodyVariables: { count: level }, level };
    },
    OrganizationCreate: (count, findNext = false) => {
        const tit = (count) => `${"OrganizationCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [2, tit(2)], [5, tit(5)], [10, tit(10)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationCreateBody", bodyVariables: { count: level }, level };
    },
    OrganizationJoin: (count, findNext = false) => {
        const tit = (count) => `${"OrganizationJoin"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "OrganizationJoinBody", bodyVariables: { count: level }, level };
    },
    PostCreate: (count, findNext = false) => {
        const tit = (count) => `${"PostCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "PostCreateBody", bodyVariables: { count: level }, level };
    },
    ProjectCreate: (count, findNext = false) => {
        const tit = (count) => `${"ProjectCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ProjectCreateBody", bodyVariables: { count: level }, level };
    },
    PullRequestCreate: (count, findNext = false) => {
        const tit = (count) => `${"PullRequestCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestCreateBody", bodyVariables: { count: level }, level };
    },
    PullRequestComplete: (count, findNext = false) => {
        const tit = (count) => `${"PullRequestComplete"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "PullRequestCompleteBody", bodyVariables: { count: level }, level };
    },
    QuestionAnswer: (count, findNext = false) => {
        const tit = (count) => `${"QuestionAnswer"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "QuestionAnswerBody", bodyVariables: { count: level }, level };
    },
    QuestionCreate: (count, findNext = false) => {
        const tit = (count) => `${"QuestionCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "QuestionCreateBody", bodyVariables: { count: level }, level };
    },
    QuizPass: (count, findNext = false) => {
        const tit = (count) => `${"QuizPass"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "QuizPassBody", bodyVariables: { count: level }, level };
    },
    ReportEnd: (count, findNext = false) => {
        const tit = (count) => `${"ReportEnd"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ReportEndBody", bodyVariables: { count: level }, level };
    },
    ReportContribute: (count, findNext = false) => {
        const tit = (count) => `${"ReportContribute"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ReportContributeBody", bodyVariables: { count: level }, level };
    },
    Reputation: (count, findNext = false) => {
        const tit = (count) => `${"ReputationPoints"}${count}Title`;
        const [level, name] = awardTier([[10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "ReputationPointsBody", bodyVariables: { count: level }, level };
    },
    RunRoutine: (count, findNext = false) => {
        const tit = (count) => `${"RunRoutine"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)], [2500, tit(2500)], [10000, tit(10000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "RunRoutineBody", bodyVariables: { count: level }, level };
    },
    RunProject: (count, findNext = false) => {
        const tit = (count) => `${"RunProject"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "RunProjectBody", bodyVariables: { count: level }, level };
    },
    RoutineCreate: (count, findNext = false) => {
        const tit = (count) => `${"RoutineCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)], [250, tit(250)], [500, tit(500)], [1000, tit(1000)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "RoutineCreateBody", bodyVariables: { count: level }, level };
    },
    SmartContractCreate: (count, findNext = false) => {
        const tit = (count) => `${"SmartContractCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "SmartContractCreateBody", bodyVariables: { count: level }, level };
    },
    StandardCreate: (count, findNext = false) => {
        const tit = (count) => `${"StandardCreate"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "StandardCreateBody", bodyVariables: { count: level }, level };
    },
    Streak: (days, findNext = false) => {
        const tit = (count) => `${"StreakDays"}${count}Title`;
        const [level, name] = awardTier([[7, tit(7)], [30, tit(30)], [100, tit(100)], [200, tit(200)], [365, tit(365)], [500, tit(500)], [750, tit(750)], [1000, tit(1000)]], days, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "StreakDaysBody", bodyVariables: { count: days }, level };
    },
    UserInvite: (count, findNext = false) => {
        const tit = (count) => `${"UserInvite"}${count}Title`;
        const [level, name] = awardTier([[1, tit(1)], [5, tit(5)], [10, tit(10)], [25, tit(25)], [50, tit(50)], [100, tit(100)]], count, findNext);
        if (!name)
            return { name: null, body: null, level: 0 };
        return { name, body: "UserInviteBody", bodyVariables: { count: level }, level };
    },
};
//# sourceMappingURL=awards.js.map