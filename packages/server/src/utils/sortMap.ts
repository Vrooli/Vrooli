const Order = {
    Asc: "asc",
    Desc: "desc",
} as const;

/**
 * Maps any object's sortBy enum to a partial Prisma query. 
 * 
 * Example: SortMap['PullRequestsAsc'] = ({ pullRequests: { _count: 'asc' } })
 * 
 * NOTE: This relies on duplicate keys always having the same query, regardless of the object. 
 * This ensures that the database implementation of various relations (bookmarks, views, etc.) is 
 * consistent across all objects.
 */
export const SortMap = {
    AmountAsc: { amount: Order.Asc },
    AmountDesc: { amount: Order.Desc },
    AnswersAsc: { answers: { _count: Order.Asc } },
    AnswersDesc: { answers: { _count: Order.Desc } },
    AttemptsAsc: { attempts: { _count: Order.Asc } },
    AttemptsDesc: { attempts: { _count: Order.Desc } },
    AttendeesAsc: { attendees: { _count: Order.Asc } },
    AttendeesDesc: { attendees: { _count: Order.Desc } },
    CalledByRoutinesAsc: { calledByRoutines: { _count: Order.Asc } },
    CalledByRoutinesDesc: { calledByRoutines: { _count: Order.Desc } },
    CommentsAsc: { comments: { _count: Order.Asc } },
    CommentsDesc: { comments: { _count: Order.Desc } },
    ComplexityAsc: { complexity: Order.Asc },
    ComplexityDesc: { complexity: Order.Desc },
    ContextSwitchesAsc: { contextSwitches: { _count: Order.Asc } },
    ContextSwitchesDesc: { contextSwitches: { _count: Order.Desc } },
    DateCompletedAsc: { completedAt: Order.Asc },
    DateCompletedDesc: { completedAt: Order.Desc },
    DateCreatedAsc: { created_at: Order.Asc },
    DateCreatedDesc: { created_at: Order.Desc },
    DateStartedAsc: { startedAt: Order.Asc },
    DateStartedDesc: { startedAt: Order.Desc },
    DateUpdatedAsc: { updated_at: Order.Asc },
    DateUpdatedDesc: { updated_at: Order.Desc },
    DueDateAsc: { dueDate: Order.Asc },
    DueDateDesc: { dueDate: Order.Desc },
    DirectoryListingsAsc: { directoryListings: { _count: Order.Asc } },
    DirectoryListingsDesc: { directoryListings: { _count: Order.Desc } },
    EndTimeAsc: { endTime: Order.Asc },
    EndTimeDesc: { endTime: Order.Desc },
    EventStartAsc: { eventStart: Order.Asc },
    EventStartDesc: { eventStart: Order.Desc },
    EventEndAsc: { eventEnd: Order.Asc },
    EventEndDesc: { eventEnd: Order.Desc },
    ForksAsc: { forks: { _count: Order.Asc } },
    ForksDesc: { forks: { _count: Order.Desc } },
    IndexAsc: { index: Order.Asc },
    IndexDesc: { index: Order.Desc },
    InvitesAsc: { invites: { _count: Order.Asc } },
    InvitesDesc: { invites: { _count: Order.Desc } },
    IssuesAsc: { issues: { _count: Order.Asc } },
    IssuesDesc: { issues: { _count: Order.Desc } },
    LabelAsc: { label: Order.Asc },
    LabelDesc: { label: Order.Desc },
    LastViewedAsc: { lastViewedAt: Order.Asc },
    LastViewedDesc: { lastViewedAt: Order.Desc },
    MeetingStartAsc: { meeting: { eventStart: Order.Asc } },
    MeetingStartDesc: { meeting: { eventStart: Order.Desc } },
    MeetingEndAsc: { meeting: { eventEnd: Order.Asc } },
    MeetingEndDesc: { meeting: { eventEnd: Order.Desc } },
    MembersAsc: { members: { _count: Order.Asc } },
    MembersDesc: { members: { _count: Order.Desc } },
    NameAsc: { name: Order.Asc },
    NameDesc: { name: Order.Desc },
    OrderAsc: { order: Order.Asc },
    OrderDesc: { order: Order.Desc },
    PeriodStartAsc: { periodStart: Order.Asc },
    PeriodStartDesc: { periodStart: Order.Desc },
    PointsEarnedAsc: { pointsEarned: Order.Asc },
    PointsEarnedDesc: { pointsEarned: Order.Desc },
    ProgressAsc: { progress: Order.Asc },
    ProgressDesc: { progress: Order.Desc },
    PullRequestsAsc: { pullRequests: { _count: Order.Asc } },
    PullRequestsDesc: { pullRequests: { _count: Order.Desc } },
    QuestionOrderAsc: { question: { order: Order.Asc } },
    QuestionOrderDesc: { question: { order: Order.Desc } },
    QuestionsAsc: { questions: { _count: Order.Asc } },
    QuestionsDesc: { questions: { _count: Order.Desc } },
    QuizzesAsc: { quizzes: { _count: Order.Asc } },
    QuizzesDesc: { quizzes: { _count: Order.Desc } },
    RecurrStartAsc: { recurrStart: Order.Asc },
    RecurrStartDesc: { recurrStart: Order.Desc },
    RecurrEndAsc: { recurrEnd: Order.Asc },
    RecurrEndDesc: { recurrEnd: Order.Desc },
    ReportsAsc: { reports: { _count: Order.Asc } },
    ReportsDesc: { reports: { _count: Order.Desc } },
    RepostsAsc: { reposts: { _count: Order.Asc } },
    RepostsDesc: { reposts: { _count: Order.Desc } },
    ResponsesAsc: { responses: { _count: Order.Asc } },
    ResponsesDesc: { responses: { _count: Order.Desc } },
    RunProjectsAsc: { runProjects: { _count: Order.Asc } },
    RunProjectsDesc: { runProjects: { _count: Order.Desc } },
    RunRoutinesAsc: { runRoutines: { _count: Order.Asc } },
    RunRoutinesDesc: { runRoutines: { _count: Order.Desc } },
    ScoreAsc: { score: Order.Asc },
    ScoreDesc: { score: Order.Desc },
    SimplicityAsc: { simplicity: Order.Asc },
    SimplicityDesc: { simplicity: Order.Desc },
    StartTimeAsc: { startTime: Order.Asc },
    StartTimeDesc: { startTime: Order.Desc },
    StepsAsc: { steps: { _count: Order.Asc } },
    StepsDesc: { steps: { _count: Order.Desc } },
    TimeTakenAsc: { timeTaken: Order.Asc },
    TimeTakenDesc: { timeTaken: Order.Desc },
    TitleAsc: { title: Order.Asc },
    TitleDesc: { title: Order.Desc },
    BookmarksAsc: { bookmarkedBy: { _count: Order.Asc } },
    BookmarksDesc: { bookmarkedBy: { _count: Order.Desc } },
    UsedForAsc: { usedFor: Order.Asc },
    UsedForDesc: { usedFor: Order.Desc },
    UserNameAsc: { user: { name: Order.Asc } },
    UserNameDesc: { user: { name: Order.Desc } },
    VersionsAsc: { versions: { _count: Order.Asc } },
    VersionsDesc: { versions: { _count: Order.Desc } },
    ViewsAsc: { viewedBy: { _count: Order.Asc } },
    ViewsDesc: { viewedBy: { _count: Order.Desc } },
    WindowEndAsc: { windowEnd: Order.Asc },
    WindowEndDesc: { windowEnd: Order.Desc },
    WindowStartAsc: { windowStart: Order.Asc },
    WindowStartDesc: { windowStart: Order.Desc },
};
