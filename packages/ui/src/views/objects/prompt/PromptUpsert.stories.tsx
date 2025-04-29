/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, Resource, ResourceUsedFor, StandardType, StandardVersion, Tag, User, endpointsStandardVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { PromptUpsert } from "./PromptUpsert.js";

// Create simplified mock data for Prompt responses
const mockPromptVersionData: StandardVersion = {
    __typename: "StandardVersion" as const,
    id: uuid(),
    comments: [],
    commentsCount: 0,
    codeLanguage: CodeLanguage.Javascript,
    created_at: new Date().toISOString(),
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
        id: uuid(),
        listFor: {
            __typename: "StandardVersion" as const,
            id: uuid(),
        } as any,
        created_at: new Date().toISOString(),
        resources: Array.from({ length: Math.floor(Math.random() * 5) + 3 }, () => ({
            __typename: "Resource" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            usedFor: ResourceUsedFor.Context,
            link: `https://example.com/resource/${Math.floor(Math.random() * 1000)}`,
            list: {} as any, // This will be set by the circular reference below
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                language: "en",
                name: `Resource ${Math.floor(Math.random() * 1000)}`,
                description: `Description for Resource ${Math.floor(Math.random() * 1000)}`,
            }],
        })) as Resource[],
        translations: [],
        updated_at: new Date().toISOString(),
    },
    root: {
        __typename: "Standard" as const,
        id: uuid(),
        isPrivate: false,
        isInternal: false,
        hasCompleteVersion: true,
        owner: { __typename: "User" as const, id: uuid() } as User,
        parent: null,
        permissions: "{}",
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["AI Prompt", "LLM", "Chatbot", "NLP", "Template", "Instruction", "Other"][Math.floor(Math.random() * 7)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        bookmarkedBy: [],
        bookmarks: 0,
        createdBy: null,
        forks: [],
        forksCount: 0,
        issues: [],
        issuesCount: 0,
        labels: [],
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
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
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
    updated_at: new Date().toISOString(),
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
    title: "Views/Objects/Prompt/PromptUpsert",
    component: PromptUpsert,
};

// Create a new Prompt
export function Create() {
    return (
        <PromptUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Prompt in a dialog
export function CreateDialog() {
    return (
        <PromptUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
CreateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing Prompt
export function Update() {
    return (
        <PromptUpsert display="page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockPromptVersionData)}/edit`,
    },
};

// Update an existing Prompt in a dialog
export function UpdateDialog() {
    return (
        <PromptUpsert
            display="Dialog"
            isCreate={false}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
UpdateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockPromptVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <PromptUpsert display="page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockPromptVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockPromptVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <PromptUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <PromptUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockPromptVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
