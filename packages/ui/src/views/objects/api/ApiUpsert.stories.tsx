import { CodeLanguage, DUMMY_ID } from "@local/shared";
import { Meta, StoryObj } from "@storybook/react";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession } from "../../../__test/storybookConsts.js";
import { SessionContext } from "../../../contexts/session.js";
import { ApiUpsert } from "./ApiUpsert.js";

const meta: Meta<typeof ApiUpsert> = {
    title: "Views/Objects/Api/ApiUpsert",
    component: ApiUpsert,
    parameters: {
        // More on how to position stories at: https://storybook.js.org/docs/react/configure/story-layout
        layout: 'fullscreen',
        msw: {
            handlers: [
                // Mock API responses for any potential backend calls
                http.get(`${API_URL}/v2/rest/apis/:id`, () => {
                    return HttpResponse.json({ data: { api: mockExistingApiData } });
                }),
                http.get(`${API_URL}/v2/rest/apiVersions/:id`, () => {
                    return HttpResponse.json({ data: { apiVersion: mockExistingApiData } });
                }),
            ],
        },
    },
    argTypes: {
        onClose: { action: 'closed' },
        onCompleted: { action: 'completed' },
        onCancel: { action: 'cancelled' },
    },
    decorators: [
        (Story, { parameters }) => (
            <SessionContext.Provider value={parameters.session || signedInNoPremiumNoCreditsSession}>
                <Story />
            </SessionContext.Provider>
        ),
    ],
};

export default meta;
type Story = StoryObj<typeof ApiUpsert>;

// --- Stories ---

export const Create: Story = {
    args: {
        display: "dialog",
        isCreate: true,
        isOpen: true,
        // onClose, onCompleted, onCancel are handled by argTypes actions
    },
    parameters: {
        session: signedInNoPremiumNoCreditsSession,
    },
};

// Create simplified mock data for API responses
const mockExistingApiData = {
    __typename: "ApiVersion",
    id: "api_version_123",
    callLink: "https://api.example.com/v1",
    directoryListings: [],
    isComplete: true,
    isPrivate: false,
    schemaLanguage: CodeLanguage.Yaml,
    schemaText: "openapi: 3.0.0\ninfo:\n  title: Mock API\n  version: 1.0.0\npaths: {}",
    versionLabel: "1.0.0",
    root: {
        __typename: "Api",
        id: "api_789",
        isPrivate: false,
        owner: { __typename: "User", id: "user_abc" },
        tags: [{ __typename: "Tag", id: "tag_1" }],
    },
    translations: [{
        __typename: "ApiVersionTranslation",
        id: DUMMY_ID,
        language: "en",
        details: "This is a mock API for demonstration.",
        name: "Mock API v1",
        summary: "A simple mock API.",
    }],
};

// This object needs to be type-compatible with PartialWithType<ApiVersion>
const mockExistingApiVersion = {
    __typename: "ApiVersion" as const,
    id: "api_version_123",
    // We don't need to include all fields since the MSW handlers will provide complete data
};

export const Update: Story = {
    args: {
        display: "dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: mockExistingApiVersion,
        // onClose, onCompleted, onCancel are handled by argTypes actions
    },
    parameters: {
        session: signedInNoPremiumNoCreditsSession,
    },
}; 