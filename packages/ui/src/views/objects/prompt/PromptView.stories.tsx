/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, type Resource, ResourceUsedFor, StandardType, type StandardVersion, type Tag, type User, endpointsStandardVersion, generatePK, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { PromptView } from "./PromptView.js";

// Create simplified mock data for Prompt responses
const mockPromptVersionData: StandardVersion = {
    __typename: "StandardVersion" as const,
    id: generatePK().toString(),
    comments: [],
    commentsCount: 0,
    codeLanguage: CodeLanguage.Javascript,
    createdAt: new Date().toISOString(),
    default: null,
    forks: [],
    forksCount: 0,
    isComplete: true,
    isDeleted: false,
    isFile: false,
    isLatest: true,
    isPrivate: false,
    props: JSON.stringify({
        inputs: [
            {
                label: "Question",
                type: "text",
                placeholder: "Type your question here...",
            },
            {
                label: "Context",
                type: "textarea",
                placeholder: "Provide any background information...",
            },
            {
                label: "Urgency",
                type: "select",
                options: ["Low", "Medium", "High"],
                default: "Medium",
            },
        ],
        output: `function generatePrompt(inputs) {
  let prompt = "";
  inputs.forEach(input => {
    if (input.value.trim() !== "") {
      prompt += prompt ? "\\n" : "";
      prompt += input.value;
    }
  });
  return prompt;
}`,
    }),
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        listFor: {
            __typename: "StandardVersion" as const,
            id: generatePK().toString(),
        } as any,
        createdAt: new Date().toISOString(),
        resources: Array.from({ length: Math.floor(Math.random() * 5) + 3 }, () => ({
            __typename: "Resource" as const,
            id: generatePK().toString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            usedFor: ResourceUsedFor.Context,
            link: `https://example.com/resource/${Math.floor(Math.random() * 1000)}`,
            list: {} as any, // This will be set by the circular reference below
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                language: "en",
                name: `Resource ${Math.floor(Math.random() * 1000)}`,
                description: `Description for Resource ${Math.floor(Math.random() * 1000)}`,
            }],
        })) as Resource[],
        translations: [],
        updatedAt: new Date().toISOString(),
    },
    root: {
        __typename: "Standard" as const,
        id: generatePK().toString(),
        isPrivate: false,
        isInternal: false,
        hasCompleteVersion: true,
        owner: { __typename: "User" as const, id: generatePK().toString() } as User,
        parent: null,
        permissions: "{}",
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: generatePK().toString(),
            tag: ["AI Prompt", "LLM", "Chatbot", "NLP", "Template", "Instruction", "Other"][Math.floor(Math.random() * 7)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        })) as Tag[],
        bookmarkedBy: [],
        bookmarks: 0,
        createdBy: null,
        forks: [],
        forksCount: 0,
        issues: [],
        issuesCount: 0,
        pullRequests: [],
        pullRequestsCount: 0,
        score: 0,
        stats: [],
        transfers: [],
        transfersCount: 0,
        translatedName: "Sample Prompt Template",
        versions: [],
        versionsCount: 1,
        views: Math.floor(Math.random() * 100_000),
        you: {
            __typename: "StandardYou",
            canBookmark: true,
            canComment: true,
            canCopy: true,
            canDelete: true,
            canReact: true,
            canRead: true,
            canTransfer: true,
            canUpdate: true,
            isBookmarked: false,
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        isDeleted: false,
    } as any,
    translations: [{
        __typename: "StandardVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a **detailed** description for this prompt template using markdown.\nExplain how to use this prompt template effectively with various LLM models.",
        name: `Customer Support Assistant Prompt v${Math.floor(Math.random() * 100)}`,
        jsonVariable: null,
    }],
    translationsCount: 1,
    updatedAt: new Date().toISOString(),
    variant: StandardType.Prompt,
    versionIndex: 1,
    versionLabel: `${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
    versionNotes: "Initial version",
    yup: null,
    you: {
        __typename: "VersionYou",
        canComment: true,
        canCopy: true,
        canDelete: true,
        canRead: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
    },
};

export default {
    title: "Views/Objects/Prompt/PromptView",
    component: PromptView,
};

export function NoResult() {
    return (
        <PromptView display="Page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <PromptView display="Page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsStandardVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockPromptVersionData)}`,
    },
};

export function SignInWithResults() {
    return (
        <PromptView display="Page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockPromptVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <PromptView display="Page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockPromptVersionData)}`,
    },
}; 
