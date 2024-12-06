//TODO!!!!
import { SessionUser } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger";
import { ExportProcessPayload } from "./queue";

async function exportAccountData(userData: SessionUser) {
    return [];
}

async function exportApisData(userData: SessionUser) {
    return [];
}

async function exportBookmarksData(userData: SessionUser) {
    return [];
}

async function exportBotsData(userData: SessionUser) {
    return [];
}

async function exportChatsData(userData: SessionUser) {
    return [];
}

async function exportCodesData(userData: SessionUser) {
    return [];
}

async function exportCommentsData(userData: SessionUser) {
    return [];
}

async function exportIssuesData(userData: SessionUser) {
    return [];
}

async function exportNotesData(userData: SessionUser) {
    return [];
}

async function exportTeamsData(userData: SessionUser) {
    return [];
}

async function exportPullRequestsData(userData: SessionUser) {
    return [];
}

async function exportProjectsData(userData: SessionUser) {
    return [];
}

async function exportQuestionsData(userData: SessionUser) {
    return [];
}

async function exportQuestionAnswersData(userData: SessionUser) {
    return [];
}

async function exportReactionsData(userData: SessionUser) {
    return [];
}

async function exportRemindersData(userData: SessionUser) {
    return [];
}

async function exportReportsData(userData: SessionUser) {
    return [];
}

async function exportRoutinesData(userData: SessionUser) {
    return [];
}

async function exportRunsData(userData: SessionUser) {
    return [];
}

async function exportSchedulesData(userData: SessionUser) {
    return [];
}

async function exportStandardsData(userData: SessionUser) {
    return [];
}

async function exportViewsData(userData: SessionUser) {
    return [];
}

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

export async function exportProcess({ data }: Job<ExportProcessPayload>) {
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
}
