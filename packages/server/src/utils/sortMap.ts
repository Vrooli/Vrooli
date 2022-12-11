const Order = {
    Asc: 'asc',
    Desc: 'desc',
} as const

/**
 * Maps any object's sortBy enum to a partial Prisma query. 
 * 
 * Example: SortMap['PullRequestsAsc'] = ({ pullRequests: { _count: 'asc' } })
 * 
 * NOTE: This relies on duplicate keys always having the same query, regardless of the object. 
 * This ensures that the database implementation of various relations (stars, views, etc.) is 
 * consistent across all objects.
 */
export const SortMap = {
    AttendeesAsc: { attendees: { _count: Order.Asc } },
    AttendeesDesc: { attendees: { _count: Order.Desc } },
    CalledByRoutinesAsc: { calledByRoutines: { _count: Order.Asc } },
    CalledByRoutinesDesc: { calledByRoutines: { _count: Order.Desc } },
    CommentsAsc: { comments: { _count: Order.Asc } },
    CommentsDesc: { comments: { _count: Order.Desc } },
    ComplexityAsc: { complexity: Order.Asc },
    ComplexityDesc: { complexity: Order.Desc },
    DateCompletedAsc: { completedAt: Order.Asc },
    DateCompletedDesc: { completedAt: Order.Desc },
    DateCreatedAsc: { created_at: Order.Asc },
    DateCreatedDesc: { created_at: Order.Desc },
    DateStartedAsc: { startedAt: Order.Asc },
    DateStartedDesc: { startedAt: Order.Desc },
    DateUpdatedAsc: { updated_at: Order.Asc },
    DateUpdatedDesc: { updated_at: Order.Desc },
    DirectoryListingsAsc: { directoryListings: { _count: Order.Asc } },
    DirectoryListingsDesc: { directoryListings: { _count: Order.Desc } },
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
    LastViewedAsc: { lastViewedAt: Order.Asc },
    LastViewedDesc: { lastViewedAt: Order.Desc },
    PullRequestsAsc: { pullRequests: { _count: Order.Asc } },
    PullRequestsDesc: { pullRequests: { _count: Order.Desc } },
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
    RunProjectsAsc: { runProjects: { _count: Order.Asc } },
    RunProjectsDesc: { runProjects: { _count: Order.Desc } },
    RunRoutinesAsc: { runRoutines: { _count: Order.Asc } },
    RunRoutinesDesc: { runRoutines: { _count: Order.Desc } },
    ScoreAsc: { score: Order.Asc },
    ScoreDesc: { score: Order.Desc },
    SimplicityAsc: { simplicity: Order.Asc },
    SimplicityDesc: { simplicity: Order.Desc },
    StarsAsc: { starredBy: { _count: Order.Asc } },
    StarsDesc: { starredBy: { _count: Order.Desc } },
    VersionsAsc: { versions: { _count: Order.Asc } },
    VersionsDesc: { versions: { _count: Order.Desc } },
    ViewsAsc: { viewedBy: { _count: Order.Asc } },
    ViewsDesc: { viewedBy: { _count: Order.Desc } },
}