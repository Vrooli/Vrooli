/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Comment, CommentThread as ThreadType, CreateIcon, lowercaseFirstLetter, uuidValidate } from "@local/shared";
import { Button, Palette, Stack, useTheme } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { CommentUpsertInput } from "components/inputs/CommentUpsertInput/CommentUpsertInput";
import { CommentThread } from "components/lists/comment";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useFindMany } from "utils/hooks/useFindMany";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
import { CommentContainerProps } from "../types";

/**
 * Common style applied when displaying this component 
 * on an object's page
 */
export const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});

export function CommentContainer({
    forceAddCommentOpen,
    isOpen,
    language,
    objectId,
    objectType,
    onAddCommentClose,
    zIndex,
}: CommentContainerProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const { t } = useTranslation();

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
    } = useFindMany<ThreadType>({
        canSearch: (params) => uuidValidate(Object.values(params.where ?? {})[0]),
        controlsUrl: false,
        searchType: "Comment",
        resolve: (result) => result.threads,
        where: {
            [`${lowercaseFirstLetter(objectType)}Id`]: objectId,
        },
    });

    /**
     * When new comment is created, add it to the list of comments
     */
    const onCommentAdd = useCallback((comment: Comment) => {
        // Make comment first, so you can see it without having to scroll to the bottom
        setAllData(curr => [{
            __typename: "CommentThread",
            comment: comment as any,
            childThreads: [],
            endCursor: null,
            totalInThread: 0,
        }, ...curr]);
    }, [setAllData]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(isMobile);
    // Show add comment input if on desktop. For mobile, we'll show a button
    useEffect(() => { setIsAddCommentOpen(!isMobile || (forceAddCommentOpen === true)); }, [forceAddCommentOpen, isMobile]);
    const handleAddCommentOpen = useCallback(() => setIsAddCommentOpen(true), []);
    const handleAddCommentClose = useCallback(() => {
        setIsAddCommentOpen(false);
        if (onAddCommentClose) onAddCommentClose();
    }, [onAddCommentClose]);

    // The add component is always visible on desktop.
    // If forceAddCommentOpen is true (i.e. parent container wants add comment to be open), 
    // then we should scroll and focus the add comment input
    useEffect(() => {
        if (!forceAddCommentOpen || isMobile) return;
        const addCommentInput = document.getElementById("markdown-input-add-comment-root");
        if (addCommentInput) {
            addCommentInput.scrollIntoView({ behavior: "smooth" });
            addCommentInput.focus();
        }
    }, [forceAddCommentOpen, isMobile]);

    return (
        <ContentCollapse isOpen={isOpen} title="Comments">
            {/* Add comment */}
            {
                isAddCommentOpen && <CommentUpsertInput
                    comment={undefined}
                    language={language}
                    objectId={objectId}
                    objectType={objectType}
                    onCancel={handleAddCommentClose}
                    onCompleted={onCommentAdd}
                    parent={null} // parent is the thread. This is a top-level comment, so no parent
                    zIndex={zIndex}
                />
            }
            {/* Sort & filter */}
            {allData.length > 0 ? <>
                <SearchButtons
                    advancedSearchParams={advancedSearchParams}
                    advancedSearchSchema={advancedSearchSchema}
                    searchType="Comment"
                    setAdvancedSearchParams={setAdvancedSearchParams}
                    setSortBy={setSortBy}
                    setTimeFrame={setTimeFrame}
                    sortBy={sortBy}
                    sortByOptions={sortByOptions}
                    timeFrame={timeFrame}
                    zIndex={zIndex}
                />
                {/* Comments list */}
                <Stack direction="column" spacing={2}>
                    {allData.map((thread, index) => (
                        <CommentThread
                            key={index}
                            canOpen={true}
                            data={thread}
                            language={language}
                            zIndex={zIndex}
                        />
                    ))}
                </Stack>
            </> : (!isAddCommentOpen && isMobile) ? <Button
                fullWidth
                startIcon={<CreateIcon />}
                onClick={handleAddCommentOpen}
                sx={{ marginTop: 2 }}
                variant="outlined"
            >{t("AddComment")}</Button> : null}
        </ContentCollapse>
    );
}
