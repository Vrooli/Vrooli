//TODO!!!!
import { SessionUser } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger";
import { ExportProcessPayload } from "./queue";

const exportAccountData = async (userData: SessionUser) => {
    return [];
};

const exportApisData = async (userData: SessionUser) => {
    return [];
};

const exportBookmarksData = async (userData: SessionUser) => {
    return [];
};

const exportBotsData = async (userData: SessionUser) => {
    return [];
};

const exportChatsData = async (userData: SessionUser) => {
    return [];
};

const exportCodesData = async (userData: SessionUser) => {
    return [];
};

const exportCommentsData = async (userData: SessionUser) => {
    return [];
};

const exportIssuesData = async (userData: SessionUser) => {
    return [];
};

const exportNotesData = async (userData: SessionUser) => {
    return [];
};

const exportTeamsData = async (userData: SessionUser) => {
    return [];
};

const exportPullRequestsData = async (userData: SessionUser) => {
    return [];
};

const exportProjectsData = async (userData: SessionUser) => {
    return [];
};

const exportQuestionsData = async (userData: SessionUser) => {
    return [];
};

const exportQuestionAnswersData = async (userData: SessionUser) => {
    return [];
};

const exportReactionsData = async (userData: SessionUser) => {
    return [];
};

const exportRemindersData = async (userData: SessionUser) => {
    return [];
};

const exportReportsData = async (userData: SessionUser) => {
    return [];
};

const exportRoutinesData = async (userData: SessionUser) => {
    return [];
};

const exportRunsData = async (userData: SessionUser) => {
    return [];
};

const exportSchedulesData = async (userData: SessionUser) => {
    return [];
};

const exportStandardsData = async (userData: SessionUser) => {
    return [];
};

const exportViewsData = async (userData: SessionUser) => {
    return [];
};

const handlerMap = {
    account: exportAccountData,
    apis: exportApisData,
    bookmarks: exportBookmarksData,
    bots: exportBotsData,
    chats: exportChatsData,
    codes: exportCodesData,
    comments: exportCommentsData,
    issues: exportIssuesData,
    notes: exportNotesData,
    teams: exportTeamsData,
    pullRequests: exportPullRequestsData,
    projects: exportProjectsData,
    questions: exportQuestionsData,
    questionAnswers: exportQuestionAnswersData,
    reactions: exportReactionsData,
    reminders: exportRemindersData,
    reports: exportReportsData,
    routines: exportRoutinesData,
    runs: exportRunsData,
    schedules: exportSchedulesData,
    standards: exportStandardsData,
    views: exportViewsData,
};

export const exportProcess = async ({ data }: Job<ExportProcessPayload>) => {
    const { flags, requestType, userData } = data;
    try {
        const results: object[] = [];

        // If not just deleting, make sure they have at least one valid email address to send link to
        //TODO

        // Iterate over each flag and call the corresponding function if set to true
        for (const [key, value] of Object.entries(flags)) {
            if (value) {
                const handlerFunction = handlerMap[key];
                if (handlerFunction) {
                    const result = await handlerFunction(userData);
                    results.push(result);
                }
            }
        }

        // Handle the requestType: Download, Delete, or both
        switch (requestType) {
            case "Download":
                // Code to bundle and send the data
                break;
            case "Delete":
                // Code to delete the data
                break;
            case "DownloadAndDelete":
                // Code to bundle, send, and then delete the data
                break;
        }

        return results;
    } catch (err) {
        logger.error("Error exporting data", { trace: "0499" });
    }
};
