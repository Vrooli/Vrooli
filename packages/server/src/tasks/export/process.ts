//TODO!!!!
import { Job } from "bull";
import { logger } from "../../events/logger";
import { SessionUserToken } from "../../types.js";
import { ExportProcessPayload } from "./queue";

const exportAccountData = async (userData: SessionUserToken) => {
    return [];
};

const exportApisData = async (userData: SessionUserToken) => {
    return [];
};

const exportBookmarksData = async (userData: SessionUserToken) => {
    return [];
};

const exportBotsData = async (userData: SessionUserToken) => {
    return [];
};

const exportChatsData = async (userData: SessionUserToken) => {
    return [];
};

const exportCodesData = async (userData: SessionUserToken) => {
    return [];
};

const exportCommentsData = async (userData: SessionUserToken) => {
    return [];
};

const exportIssuesData = async (userData: SessionUserToken) => {
    return [];
};

const exportNotesData = async (userData: SessionUserToken) => {
    return [];
};

const exportTeamsData = async (userData: SessionUserToken) => {
    return [];
};

const exportPullRequestsData = async (userData: SessionUserToken) => {
    return [];
};

const exportProjectsData = async (userData: SessionUserToken) => {
    return [];
};

const exportQuestionsData = async (userData: SessionUserToken) => {
    return [];
};

const exportQuestionAnswersData = async (userData: SessionUserToken) => {
    return [];
};

const exportReactionsData = async (userData: SessionUserToken) => {
    return [];
};

const exportRemindersData = async (userData: SessionUserToken) => {
    return [];
};

const exportReportsData = async (userData: SessionUserToken) => {
    return [];
};

const exportRoutinesData = async (userData: SessionUserToken) => {
    return [];
};

const exportRunsData = async (userData: SessionUserToken) => {
    return [];
};

const exportSchedulesData = async (userData: SessionUserToken) => {
    return [];
};

const exportStandardsData = async (userData: SessionUserToken) => {
    return [];
};

const exportViewsData = async (userData: SessionUserToken) => {
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
