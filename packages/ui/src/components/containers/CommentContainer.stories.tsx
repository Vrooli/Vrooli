import { Comment, CommentFor, CommentThread, CommentTranslation, endpointsComment, uuid } from "@local/shared";
import { Box, Paper, useTheme } from "@mui/material";
import { HttpResponse, http } from "msw";
import { API_URL, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { ScrollBox } from "../../styles.js";
import { Navbar } from "../navigation/Navbar.js";
import { CommentContainer } from "./CommentContainer.js";

/**
 * CommentContainer - A component that manages comments for various object types.
 * Displays a comment input and thread list with support for pagination, sorting and filtering.
 */

// Container styles for better display
const outerBoxStyle = {
    width: "100%",
    minHeight: "400px",
    padding: "20px",
    maxWidth: "800px",
    margin: "0 auto",
} as const;

const containerStyle = {
    padding: "16px",
    borderRadius: "16px",
} as const;

// Wrapper component that simulates how the CommentContainer would appear in a real page
function Outer({ children }: { children: React.ReactNode }) {
    const { palette } = useTheme();
    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title="Comments" />
                <Box sx={outerBoxStyle}>
                    <Paper
                        elevation={2}
                        sx={{
                            ...containerStyle,
                            backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e"
                        }}
                    >
                        {children}
                    </Paper>
                </Box>
            </ScrollBox>
        </PageContainer>
    );
}

// Mock data creators
interface MockCommentTranslation extends Partial<CommentTranslation> {
    __typename: "CommentTranslation";
    id: string;
    language: string;
    text: string;
}

interface MockCommentOwner {
    __typename: "User";
    id: string;
    handle: string;
    name: string;
}

interface MockCommentYou {
    __typename: "CommentYou";
    canDelete: boolean;
    canUpdate: boolean;
    canReply: boolean;
    canReport: boolean;
    canBookmark: boolean;
    canReact: boolean;
    isBookmarked: boolean;
    reaction: string | null;
}

interface MockComment extends Partial<Comment> {
    __typename: "Comment";
    id: string;
    created_at: string;
    updated_at: string;
    translations: MockCommentTranslation[];
    owner: MockCommentOwner;
    you: MockCommentYou;
}

interface MockCommentThread extends Partial<CommentThread> {
    __typename: "CommentThread";
    comment: MockComment;
    childThreads: MockCommentThread[];
    endCursor: string | null;
    totalInThread: number;
}

const createMockComment = (id: string, text: string, owner: string = "John Doe", isCurrentUser: boolean = false): MockComment => ({
    __typename: "Comment" as const,
    id,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    translations: [{
        __typename: "CommentTranslation" as const,
        id: `${id}-translation`,
        language: "en",
        text,
    }],
    owner: {
        __typename: "User" as const,
        id: isCurrentUser ? "current-user-id" : `user-${id}`,
        handle: owner,
        name: owner,
    },
    you: {
        __typename: "CommentYou" as const,
        canDelete: isCurrentUser || owner === "Maria Garcia", // Only allow deletion for current user or specific moderator
        canUpdate: isCurrentUser, // Only allow updates for current user
        canReply: true, // Everyone can reply
        canReport: !isCurrentUser, // Can't report your own comments
        canBookmark: true,
        canReact: true,
        isBookmarked: false,
        reaction: null,
    },
});

const createMockThread = (comment: MockComment, childThreads: MockCommentThread[] = []): MockCommentThread => ({
    __typename: "CommentThread" as const,
    comment,
    childThreads,
    endCursor: null,
    totalInThread: childThreads.length,
});

// Mock comment data
const mockComments = [
    createMockThread(
        createMockComment(uuid(), "This is a top-level comment that discusses the main features of this object. It's quite detailed and provides useful information.", "Current User", true),
        [
            createMockThread(
                createMockComment(uuid(), "Great point! I'd like to add that there are additional considerations here.", "Jane Smith")
            ),
            createMockThread(
                createMockComment(uuid(), "I disagree with some parts of your assessment. Let me explain why in this long message.\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.", "Alex Johnson")
            ),
        ]
    ),
    createMockThread(
        createMockComment(uuid(), "Has anyone tried using this with the new API? I'm getting some unexpected results.", "Sam Wilson")
    ),
    createMockThread(
        createMockComment(uuid(), "The documentation for this could be improved. Here are some suggestions.", "Maria Garcia")
    ),
];

// Mock comment responses
const emptyCommentsResponse = {
    threads: [] as MockCommentThread[],
    totalCount: 0,
};

const populatedCommentsResponse = {
    threads: mockComments,
    totalCount: mockComments.length,
};

// Base component props
const defaultProps = {
    language: "en",
    objectId: uuid(),
    objectType: CommentFor.CodeVersion,
};

// Sample comment handlers for MSW
const commentHandlers = (responseData = emptyCommentsResponse) => [
    // Mock find many comments endpoint
    http.get(`${API_URL}/v2/rest${endpointsComment.findMany.endpoint}*`, () => {
        return HttpResponse.json({ data: responseData });
    }),

    // Mock create comment endpoint
    http.post(`${API_URL}/v2/rest${endpointsComment.createOne.endpoint}`, async ({ request }) => {
        const body = await request.json() as Record<string, any>;
        console.log("Create comment request body:", body);

        // Check for text in the first translation
        let commentText = "New comment";
        if (body.text) {
            // Handle direct text property
            commentText = body.text;
        } else if (body.translations && Array.isArray(body.translations) && body.translations.length > 0) {
            // Handle translations array
            commentText = body.translations[0].text || "New comment";
        }

        // Create a mock response with the submitted data
        const newComment = createMockComment(
            uuid(),
            commentText,
            "Current User",
            true // Mark as current user's comment
        );

        return HttpResponse.json({ data: newComment });
    }),
];

// Loading handlers that delay response
const loadingHandlers = [
    http.get(`${API_URL}/v2/rest${endpointsComment.findMany.endpoint}*`, async () => {
        // Simulate a delay
        await new Promise(resolve => setTimeout(resolve, 2000));
        return HttpResponse.json({ data: emptyCommentsResponse });
    }),
];

// Read-only handlers that return 403 for create operations
const readOnlyHandlers = (responseData = emptyCommentsResponse) => [
    // Allow read operations
    http.get(`${API_URL}/v2/rest${endpointsComment.findMany.endpoint}*`, () => {
        return HttpResponse.json({ data: responseData });
    }),

    // Deny write operations
    http.post(`${API_URL}/v2/rest${endpointsComment.createOne.endpoint}`, () => {
        return HttpResponse.json(
            {
                success: false,
                error: "You don't have permission to create comments"
            },
            { status: 403 }
        );
    }),
];

export default {
    title: "Components/Containers/CommentContainer",
    component: CommentContainer,
    decorators: [
        (Story) => <Outer><Story /></Outer>
    ],
    parameters: {
        docs: {
            description: {
                component: "Displays comments for an object with thread functionality and comment input."
            }
        },
        session: signedInPremiumWithCreditsSession,
    }
};

// Story Components
export function EmptyWithComments() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
EmptyWithComments.parameters = {
    msw: {
        handlers: commentHandlers(emptyCommentsResponse)
    },
    docs: {
        description: {
            story: "Empty state with ability to leave comments."
        }
    }
};

export function EmptyReadOnly() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
EmptyReadOnly.parameters = {
    msw: {
        handlers: readOnlyHandlers(emptyCommentsResponse)
    },
    docs: {
        description: {
            story: "Empty state without ability to leave comments (read-only)."
        }
    }
};

export function PopulatedWithComments() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
PopulatedWithComments.parameters = {
    msw: {
        handlers: commentHandlers(populatedCommentsResponse)
    },
    docs: {
        description: {
            story: "Populated state with ability to leave comments. Shows a realistic mix of comments with different permissions: only your own comments can be edited, and only your comments or those from the moderator (Maria Garcia) can be deleted."
        }
    }
};

export function PopulatedReadOnly() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
PopulatedReadOnly.parameters = {
    msw: {
        handlers: readOnlyHandlers(populatedCommentsResponse)
    },
    docs: {
        description: {
            story: "Populated state without ability to leave comments (read-only). Same permission structure as PopulatedWithComments, but API returns 403 for create operations."
        }
    }
};

export function Loading() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
Loading.parameters = {
    msw: {
        handlers: loadingHandlers
    },
    docs: {
        description: {
            story: "Loading state while fetching comments."
        }
    }
};

export function MobileView() {
    return (
        <CommentContainer
            {...defaultProps}
        />
    );
}
MobileView.parameters = {
    msw: {
        handlers: commentHandlers(populatedCommentsResponse)
    },
    viewport: {
        defaultViewport: 'mobile1',
    },
    docs: {
        description: {
            story: "Mobile view (narrow width) with comments."
        }
    }
};

export function ForceAddCommentOpen() {
    return (
        <CommentContainer
            {...defaultProps}
            forceAddCommentOpen={true}
        />
    );
}
ForceAddCommentOpen.parameters = {
    msw: {
        handlers: commentHandlers(emptyCommentsResponse)
    },
    docs: {
        description: {
            story: "Comments with add comment form forced open."
        }
    }
}; 