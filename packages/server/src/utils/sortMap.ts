const SortOrder = {
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
    CommentsAsc: { comments: { _count: SortOrder.Asc } },
    CommentsDesc: { comments: { _count: SortOrder.Desc } },
    ComplexityAsc: { complexity: SortOrder.Asc },
    ComplexityDesc: { complexity: SortOrder.Desc },
    DateCompletedAsc: { completedAt: SortOrder.Asc },
    DateCompletedDesc: { completedAt: SortOrder.Desc },
    DateCreatedAsc: { created_at: SortOrder.Asc },
    DateCreatedDesc: { created_at: SortOrder.Desc },
    DateStartedAsc: { startedAt: SortOrder.Asc },
    DateStartedDesc: { startedAt: SortOrder.Desc },
    DateUpdatedAsc: { updated_at: SortOrder.Asc },
    DateUpdatedDesc: { updated_at: SortOrder.Desc },
    DirectoryListingsAsc: { directoryListings: { _count: SortOrder.Asc } },
    DirectoryListingsDesc: { directoryListings: { _count: SortOrder.Desc } },
    ForksAsc: { forks: { _count: SortOrder.Asc } },
    ForksDesc: { forks: { _count: SortOrder.Desc } },
    IndexAsc: { index: SortOrder.Asc },
    IndexDesc: { index: SortOrder.Desc },
    IssuesAsc: { issues: { _count: SortOrder.Asc } },
    IssuesDesc: { issues: { _count: SortOrder.Desc } },
    LastViewedAsc: { lastViewed: SortOrder.Asc },
    LastViewedDesc: { lastViewed: SortOrder.Desc },
    PullRequestsAsc: { pullRequests: { _count: SortOrder.Asc } },
    PullRequestsDesc: { pullRequests: { _count: SortOrder.Desc } },
    QuestionsAsc: { questions: { _count: SortOrder.Asc } },
    QuestionsDesc: { questions: { _count: SortOrder.Desc } },
    QuizzesAsc: { quizzes: { _count: SortOrder.Asc } },
    QuizzesDesc: { quizzes: { _count: SortOrder.Desc } },
    ReportsAsc: { reports: { _count: SortOrder.Asc } },
    ReportsDesc: { reports: { _count: SortOrder.Desc } },
    RunProjectsAsc: { runProjects: { _count: SortOrder.Asc } },
    RunProjectsDesc: { runProjects: { _count: SortOrder.Desc } },
    RunRoutinesAsc: { runRoutines: { _count: SortOrder.Asc } },
    RunRoutinesDesc: { runRoutines: { _count: SortOrder.Desc } },
    ScoreAsc: { score: SortOrder.Asc },
    ScoreDesc: { score: SortOrder.Desc },
    SimplicityAsc: { simplicity: SortOrder.Asc },
    SimplicityDesc: { simplicity: SortOrder.Desc },
    StarsAsc: { starredBy: { _count: SortOrder.Asc } },
    StarsDesc: { starredBy: { _count: SortOrder.Desc } },
    VersionsAsc: { versions: { _count: SortOrder.Asc } },
    VersionsDesc: { versions: { _count: SortOrder.Desc } },
    ViewsAsc: { viewedBy: { _count: SortOrder.Asc } },
    ViewsDesc: { viewedBy: { _count: SortOrder.Desc } },
}