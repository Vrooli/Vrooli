/**
 * CommentContainer - A component that manages comments for various object types.
 * Displays a comment input and thread list with support for pagination, sorting and filtering.
 */
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { endpointsComment, lowercaseFirstLetter, validatePK, type Comment, type CommentCreateInput, type CommentThread as ThreadType } from "@vrooli/shared";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { SearchButtons } from "../buttons/SearchButtons.js";
import { AdvancedInputBase } from "../inputs/AdvancedInput/AdvancedInput.js";
import { DEFAULT_FEATURES } from "../inputs/AdvancedInput/utils.js";
import { CommentThread } from "../lists/CommentThread/CommentThread.js";
import { type CommentContainerProps } from "./types.js";

// Configure comment input features
const COMMENT_INPUT_FEATURES = {
    ...DEFAULT_FEATURES,
    allowFileAttachments: false,
    allowImageAttachments: false,
    allowTextAttachments: false,
    allowContextDropdown: false,
    allowTasks: false,
    allowVoiceInput: false,
    allowExpand: true,
    allowFormatting: true,
    allowCharacterLimit: true,
    allowSubmit: true,
    allowSpellcheck: true,
};

export function CommentContainer({
    forceAddCommentOpen = false,
    language,
    objectId,
    objectType,
    onAddCommentClose,
}: CommentContainerProps) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const { t } = useTranslation();

    // State for the comment input
    const [commentText, setCommentText] = useState<string>("");
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

    // State for thread data from API
    const {
        advancedSearchParams,
        advancedSearchSchema,
        allData,
        setAdvancedSearchParams,
        setAllData,
        setSortBy,
        setTimeFrame,
        sortBy,
        sortByOptions,
        timeFrame,
        // Use the loading state from the API response if available, otherwise default to false
        isLoading = false,
    } = useFindMany<ThreadType>({
        canSearch: () => validatePK(objectId),
        controlsUrl: false,
        searchType: "Comment",
        resolve: (result) => result.threads,
        where: {
            [`${lowercaseFirstLetter(objectType)}Id`]: objectId,
        },
    });

    // Fetch for creating new comments
    const [createCommentMutation, { loading: isCreateLoading }] = useLazyFetch<CommentCreateInput, Comment>(endpointsComment.createOne);

    // Handle changes to comment text
    const handleCommentChange = useCallback((value: string) => {
        setCommentText(value);
    }, []);

    // Handle comment submission
    const handleSubmitComment = useCallback(() => {
        if (!commentText.trim() || isSubmitting || isCreateLoading) return;

        setIsSubmitting(true);

        const input: CommentCreateInput = {
            [`${lowercaseFirstLetter(objectType)}Id`]: objectId,
            translations: [{
                language,
                text: commentText,
            }],
        };

        fetchLazyWrapper<CommentCreateInput, Comment>({
            fetch: createCommentMutation,
            inputs: input,
            successCondition: (data) => !!data?.id,
            onSuccess: (newComment) => {
                // Add the new comment to the thread list
                setAllData(curr => [{
                    __typename: "CommentThread",
                    comment: newComment,
                    childThreads: [],
                    endCursor: null,
                    totalInThread: 0,
                }, ...curr]);

                // Clear input
                setCommentText("");

                // If on mobile, notify parent that we've added a comment
                if (isMobile && onAddCommentClose) {
                    onAddCommentClose();
                }
            },
            errorMessage: () => ({ messageKey: "CreateCommentFailed", defaultValue: "Failed to create comment" }),
            onCompleted: () => setIsSubmitting(false),
        });
    }, [commentText, createCommentMutation, isMobile, isCreateLoading, isSubmitting, language, objectId, objectType, onAddCommentClose, setAllData]);

    // Determine if there are comments to display
    const hasComments = useMemo(() => allData.length > 0, [allData.length]);

    // Message to show when no comments exist
    const emptyStateMessage = useMemo(() => (
        <Box
            sx={{
                py: 4,
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                minHeight: "120px",
                color: palette.text.secondary,
            }}
        >
            <IconCommon
                decorative
                name="Comment"
                size={40}
                fill={palette.divider}
                style={{ marginBottom: "16px", opacity: 0.7 }}
            />
            <Typography variant="body1" fontWeight="medium">
                {t("NoCommentsYet", { defaultValue: "No comments yet" })}
            </Typography>
            <Typography variant="body2" sx={{ mt: 1 }}>
                {t("BeTheFirstToComment", { defaultValue: "Be the first to leave a comment" })}
            </Typography>
        </Box>
    ), [palette.divider, palette.text.secondary, t]);

    return (
        <Box>
            {/* Comment input section */}
            <Box sx={{
                border: `1px solid ${palette.divider}`,
                borderRadius: "24px",
            }}>
                <AdvancedInputBase
                    name="commentText"
                    value={commentText}
                    onChange={handleCommentChange}
                    onSubmit={handleSubmitComment}
                    maxChars={16_256}
                    placeholder={t("AddComment", { defaultValue: "Add a comment..." })}
                    features={COMMENT_INPUT_FEATURES}
                    disabled={isSubmitting || isCreateLoading}
                />
            </Box>

            <Stack direction="column" spacing={2}>
                {/* Loading state */}
                {isLoading && (
                    <Box sx={{ py: 2 }}>
                        <Alert severity="info">{t("LoadingComments", { defaultValue: "Loading comments..." })}</Alert>
                    </Box>
                )}

                {/* Empty state */}
                {!isLoading && !hasComments && emptyStateMessage}

                {/* Comments with sorting tools */}
                {hasComments && (
                    <Box pt={2}>
                        <SearchButtons
                            advancedSearchParams={advancedSearchParams}
                            advancedSearchSchema={advancedSearchSchema}
                            controlsUrl={false}
                            searchType="Comment"
                            setAdvancedSearchParams={setAdvancedSearchParams}
                            setSortBy={setSortBy}
                            setTimeFrame={setTimeFrame}
                            sortBy={sortBy}
                            sortByOptions={sortByOptions}
                            timeFrame={timeFrame}
                        />

                        {/* Comment thread list */}
                        <Stack direction="column" spacing={3} sx={{ mt: 2 }}>
                            {allData.map((thread, index) => (
                                <CommentThread
                                    key={thread.comment?.id || index}
                                    canOpen={true}
                                    data={thread}
                                    language={language}
                                />
                            ))}
                        </Stack>
                    </Box>
                )}
            </Stack>
        </Box>
    );
}
