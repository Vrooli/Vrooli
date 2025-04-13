/* eslint-disable no-magic-numbers */
import { Api, ApiVersion, CodeLanguage, Resource, ResourceList, ResourceUsedFor, Tag, User, endpointsApiVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ApiView } from "./ApiView.js";

// Create simplified mock data for API responses
const mockApiVersionData: ApiVersion = {
    __typename: "ApiVersion" as const,
    id: uuid(),
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    callLink: "https://reddit.com/v1",
    documentationLink: "https://docs.example.com/v1",
    directoryListings: [],
    isComplete: true,
    isPrivate: false,
    resourceList: {
        __typename: "ResourceList" as const,
        id: uuid(),
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
        })) as unknown as Resource[], // Use unknown to bypass type checking until runtime
        translations: [],
        updated_at: new Date().toISOString(),
    } as unknown as ResourceList,
    schemaLanguage: CodeLanguage.Yaml,
    schemaText: `openapi: 3.0.0
info:
  title: Mock API
  version: ${Math.floor(Math.random() * 1000)}
paths:
  /comments:
    get:
      summary: List all comments
      description: Returns a list of comments
      parameters:
        - name: limit
          in: query
          description: Maximum number of items to return
          required: false
          schema:
            type: integer
            format: int32
            default: 20
        - name: offset
          in: query
          description: Number of items to skip
          required: false
          schema:
            type: integer
            format: int32
            default: 0
      responses:
        '200':
          description: A list of comments
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Comment'
    post:
      summary: Create a new comment
      description: Creates a new comment in the database
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CommentInput'
      responses:
        '201':
          description: Comment created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Comment'
        '400':
          description: Invalid input data
  /comments/{id}:
    get:
      summary: Get comment by ID
      description: Returns a single comment
      parameters:
        - name: id
          in: path
          description: Comment ID
          required: true
          schema:
            type: string
      responses:
        '200':
          description: A comment object
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Comment'
        '404':
          description: Comment not found
    put:
      summary: Update a comment
      description: Updates an existing comment
      parameters:
        - name: id
          in: path
          description: Comment ID
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CommentInput'
      responses:
        '200':
          description: Comment updated successfully
        '404':
          description: Comment not found
    delete:
      summary: Delete a comment
      description: Deletes an existing comment
      parameters:
        - name: id
          in: path
          description: Comment ID
          required: true
          schema:
            type: string
      responses:
        '204':
          description: Comment deleted successfully
        '404':
          description: Comment not found
components:
  schemas:
    Comment:
      type: object
      properties:
        id:
          type: string
        text:
          type: string
        author:
          type: string
        createdAt:
          type: string
          format: date-time
    CommentInput:
      type: object
      properties:
        text:
          type: string
        author:
          type: string
      required:
        - text
        - author`,
    versionLabel: `${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    root: {
        __typename: "Api" as const,
        id: uuid(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() } as unknown as User,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["Automation", "AI Agents", "Software Development", "Agriculture", "Healthcare", "Finance", "Education", "Government", "Retail", "Manufacturing", "Energy", "Transportation", "Entertainment", "Other"][Math.floor(Math.random() * 14)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        versions: [],
        views: Math.floor(Math.random() * 100_000),
    } as unknown as Api,
    translations: [{
        __typename: "ApiVersionTranslation" as const,
        id: uuid(),
        language: "en",
        details: "This is a **detailed** description for the mock API using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
        name: `Mock API v${Math.floor(Math.random() * 1000)}`,
        summary: "A simple mock API.",
    }],
};

export default {
    title: "Views/Objects/Api/ApiView",
    component: ApiView,
};

export function NoResult() {
    return (
        <ApiView display="page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <ApiView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}`,
    },
};

export function SignInWithResults() {
    return (
        <ApiView display="page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <ApiView display="page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}`,
    },
};

export function Own() {
    return (
        <ApiView display="page" />
    );
}
Own.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, () => {
                // Create a modified version of the mock data with owner permissions
                const mockWithOwnerPermissions = {
                    ...mockApiVersionData,
                    root: {
                        ...mockApiVersionData.root,
                        you: {
                            // Full permissions as the owner
                            canBookmark: true,
                            canDelete: true,
                            canUpdate: true,
                            canRead: true,
                            isBookmarked: false,
                            isOwner: true,
                            isStarred: false,
                        }
                    },
                    permissions: {
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    }
                };

                return HttpResponse.json({ data: mockWithOwnerPermissions });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}`,
    },
};
