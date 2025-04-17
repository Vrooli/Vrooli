/* eslint-disable no-magic-numbers */
import { Chat, ChatCreateInput, ChatInvite, ChatMessage, ChatParticipant, ChatUpdateInput, ChatYou, DUMMY_ID, Label, User, uuid, uuidToBase36 } from "@local/shared";
import type { Meta, StoryObj } from "@storybook/react";
import { waitFor, within } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { expect } from "chai";
import { HttpResponse, http } from "msw";
import { signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { SessionContext } from "../../../contexts/session.js";
import { ChatCrud, VALYXA_INFO } from "./ChatCrud.js";
import { ChatCrudProps } from "./types.js";

// Ensure VALYXA_INFO conforms to User type
const valyxaUser: User = {
    __typename: "User",
    id: VALYXA_INFO.id,
    name: VALYXA_INFO.name,
    isBot: VALYXA_INFO.isBot,
    emails: [],
    // picture: null, // Removed
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    roles: [],
    settings: [],
    preferences: [],
    permissions: { canCreate: {}, canRead: {}, canUpdate: {}, canDelete: {} },
};

// Mock chat data
const mockChatId = uuid();
const mockBase36ChatId = uuidToBase36(mockChatId);
const mockNewChatId = uuid();

// Mock ChatYou type - Assuming it links via `chat` and `user` objects/IDs
const mockChatYou: ChatYou = {
    __typename: "ChatYou",
    // chatId: mockChatId, // Removed as it's likely derived or not present directly
    userId: signedInPremiumWithCreditsSession.users[0].id,
    chat: { id: mockChatId } as Chat, // Link ChatYou to Chat
    user: { id: signedInPremiumWithCreditsSession.users[0].id } as User, // Link ChatYou to User
    lastReadMessageId: null,
    lastReadMessageAt: null,
};

// Function to create mock participants using standard function declaration
function createMockParticipant(user: User, chatId: string): ChatParticipant {
    return {
        __typename: "ChatParticipant",
        id: uuid(),
        user,
        chat: { id: chatId } as Chat,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
    };
}

// Function to create mock messages using standard function declaration and 'text' field
function createMockMessage(user: User, chatId: string, messageText: string): ChatMessage {
    return {
        __typename: "ChatMessage",
        id: uuid(),
        chat: { id: chatId } as Chat,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        text: messageText, // Use 'text' instead of 'content'
        user,
        parentMessageId: null,
        status: "sent",
        type: "text",
        children: [],
        contexts: [],
    };
}

const mockExistingChat: Chat = {
    __typename: "Chat",
    id: mockChatId,
    openToAnyoneWithInvite: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    team: null, // Use team object/null, not teamId
    invites: [] as ChatInvite[],
    labels: [] as Label[],
    participants: [
        createMockParticipant(signedInPremiumWithCreditsSession.users[0], mockChatId), // Use the SessionUser
        createMockParticipant(valyxaUser, mockChatId),
    ],
    messages: [
        createMockMessage(valyxaUser, mockChatId, "Hello from Valyxa!"),
    ],
    translations: [{
        __typename: "ChatTranslation",
        id: uuid(),
        language: "en",
        name: "Existing Chat",
        description: "This is an existing chat.",
    }],
    invitesCount: 0,
    labelsCount: 0,
    participantsCount: 2,
    restrictedToRoles: [],
    you: { ...mockChatYou, chat: { id: mockChatId } as Chat, user: { id: signedInPremiumWithCreditsSession.users[0].id } as User }, // Ensure ChatYou links to this chat
    // teamId removed
};

const mockCreatedChat: Chat = {
    __typename: "Chat",
    id: mockNewChatId,
    openToAnyoneWithInvite: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    team: null,
    invites: [],
    labels: [],
    participants: [
        createMockParticipant(signedInPremiumWithCreditsSession.users[0], mockNewChatId),
        createMockParticipant(valyxaUser, mockNewChatId),
    ],
    messages: [],
    translations: [{
        __typename: "ChatTranslation",
        id: uuid(),
        language: "en",
        name: "New Chat",
        description: "",
    }],
    invitesCount: 0,
    labelsCount: 0,
    participantsCount: 2,
    restrictedToRoles: [],
    you: { ...mockChatYou, chat: { id: mockNewChatId } as Chat, user: { id: signedInPremiumWithCreditsSession.users[0].id } as User }, // Adjust ChatYou for new chat
    // teamId removed
};


const meta: Meta<typeof ChatCrud> = {
    title: "Views/Objects/Chat/ChatCrud",
    component: ChatCrud,
    // Args defined in meta should be truly default or optional
    args: {
        display: "page",
        // Provide default function for onClose, assuming it's required or commonly used
        // Note: If onClose is truly optional in ChatCrudProps, this might not be needed here
        // but stories often need a mock handler.
        onClose: () => console.log("Default onClose called"),
    },
    parameters: {
        layout: "fullscreen",
        msw: {
            handlers: [
                // Mock findOne for existing chat
                http.get(`/api/chats/${mockBase36ChatId}`, () => {
                    return HttpResponse.json({ data: mockExistingChat });
                }),
                // Mock findOne for non-existent chat (triggers create flow)
                http.get(`/api/chats/${DUMMY_ID}`, () => {
                    return HttpResponse.json({ data: null });
                }),
                http.get("/api/chats/add", () => {
                    return HttpResponse.json({ data: null });
                }),
                // Mock createOne
                http.post("/api/chats", async ({ request }) => {
                    const body = await request.json() as ChatCreateInput;
                    const newChatId = body.id ?? mockNewChatId;
                    const createdChatResponse: Chat = {
                        ...mockCreatedChat,
                        id: newChatId,
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        // Ensure ChatYou links correctly
                        you: { ...mockCreatedChat.you, chat: { id: newChatId } as Chat },
                        translations: body.translationsCreate?.map(t => ({
                            ...t,
                            __typename: "ChatTranslation",
                            id: t.id ?? uuid(),
                        })) ?? mockCreatedChat.translations,
                    };
                    return HttpResponse.json({ data: createdChatResponse }, { status: 201 });
                }),
                // Mock updateOne
                http.put(`/api/chats/${mockBase36ChatId}`, async ({ request }) => {
                    const body = await request.json() as ChatUpdateInput;
                    const updatedChat: Chat = {
                        ...mockExistingChat,
                        updated_at: new Date().toISOString(),
                        openToAnyoneWithInvite: body.openToAnyoneWithInvite ?? mockExistingChat.openToAnyoneWithInvite,
                        translations: body.translationsUpdate
                            ? mockExistingChat.translations?.map(t => {
                                const updateData = body.translationsUpdate?.find(u => u.id === t.id);
                                return updateData ? { ...t, ...updateData } : t;
                            })
                            : mockExistingChat.translations,
                        // Update `you` field if relevant properties change (e.g., lastReadMessage)
                    };
                    return HttpResponse.json({ data: updatedChat });
                }),
                // Mock user search
                http.get("/api/users", () => {
                    return HttpResponse.json({
                        // Return SessionUser in search results if appropriate
                        data: [signedInPremiumWithCreditsSession.users[0], valyxaUser],
                        total: 2,
                    });
                }),
                // Mock websocket endpoints
                http.get("/socket.io/*", () => {
                    return new HttpResponse(null, { status: 404 });
                }),
                http.post("/socket.io/*", () => {
                    return new HttpResponse(null, { status: 404 });
                }),
            ],
        },
        // Route is now defined per-story
    },
    tags: ["autodocs"],
    decorators: [
        (Story) => (
            <SessionContext.Provider value={signedInPremiumWithCreditsSession}>
                <Story />
            </SessionContext.Provider>
        ),
    ],
};

export default meta;
// Use the imported ChatCrudProps for explicit typing
type Story = StoryObj<ChatCrudProps>; // Use the imported props type

/** Story for creating a new chat instance. */
export const NewChat: Story = {
    args: {
        // Explicitly type args if needed, or rely on StoryObj<ChatCrudProps>
        // args: { // Example of explicit typing if needed
        // } as Partial<ChatCrudProps>,
        display: "page",
        isCreate: true,
        isOpen: true, // Keep isOpen
        onClose: () => console.log("NewChat closed"), // Keep onClose
        // overrideObject is undefined for create, so no need to specify unless overriding a default
    },
    parameters: {
        route: {
            path: "/chat/add", // Mock the route for create
        },
    },
    // Decorators can be inherited or overridden if needed
    play: async ({ canvasElement, step }) => {
        const canvas = within(canvasElement);

        await step("Wait for initial chat elements to render", async () => {
            await canvas.findByPlaceholderText(/Message @Valyxa/i);
            // Use chai assertion style
            await waitFor(() => expect(canvas.queryByDisplayValue("New Chat")).to.exist);
        });

        await step("Type and send a message", async () => {
            const messageInput = canvas.getByPlaceholderText(/Message @Valyxa/i);
            await userEvent.type(messageInput, "Hello, Valyxa!", { delay: 50 });
        });
    },
};

/** Story for viewing/editing an existing chat instance. */
export const ExistingChat: Story = {
    args: {
        // Explicitly type args if needed
        // args: { // Example
        // } as Partial<ChatCrudProps>,
        display: "page",
        isCreate: false,
        isOpen: true, // Keep isOpen
        overrideObject: mockExistingChat.id, // Keep overrideObject
        onClose: () => console.log("ExistingChat closed"), // Keep onClose
    },
    parameters: {
        route: {
            path: `/chat/${mockBase36ChatId}`, // Mock the route for existing chat
        },
    },
    play: async ({ canvasElement, step }) => {
        const canvas = within(canvasElement);

        await step("Wait for existing chat to load", async () => {
            await canvas.findByDisplayValue("Existing Chat");
            await canvas.findByText("Hello from Valyxa!");
        });

        await step("Edit the chat title", async () => {
            const titleDisplay = canvas.getByDisplayValue("Existing Chat");
            await userEvent.click(titleDisplay);
            const titleInput = await canvas.findByRole("textbox", { name: /Name/i });
            await userEvent.clear(titleInput);
            await userEvent.type(titleInput, "Updated Chat Title", { delay: 50 });
            await userEvent.tab();
            // Use chai assertion style
            await waitFor(() => expect(canvas.queryByDisplayValue("Existing Chat")).not.to.exist);
            await canvas.findByDisplayValue("Updated Chat Title");
        });

        await step("Type and send a reply", async () => {
            const messageInput = canvas.getByPlaceholderText(/Message @Valyxa/i);
            await userEvent.type(messageInput, "Replying to Valyxa!", { delay: 50 });
        });
    },
};

// Add more stories as needed 
