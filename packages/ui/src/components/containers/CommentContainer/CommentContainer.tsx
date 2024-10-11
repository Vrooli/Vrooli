/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Comment, CommentThread as ThreadType, lowercaseFirstLetter, uuidValidate } from "@local/shared";
import { Button, Stack, useTheme } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { CommentThread } from "components/lists/CommentThread/CommentThread";
import { useFindMany } from "hooks/useFindMany";
import { useWindowSize } from "hooks/useWindowSize";
import { CreateIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { CommentUpsert } from "views/objects/comment";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
import { CommentContainerProps } from "../types";

export function CommentContainer({
    forceAddCommentOpen,
    isOpen,
    language,
    objectId,
    objectType,
    onAddCommentClose,
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
        canSearch: () => uuidValidate(objectId),
        controlsUrl: false,
        searchType: "Comment",
        resolve: (result) => result.threads,
        where: {
            [`${lowercaseFirstLetter(objectType)}Id`]: objectId,
        },
    });

    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(!isMobile);
    // Show add comment input if on desktop. For mobile, we'll show a button
    useEffect(() => { setIsAddCommentOpen(!isMobile || (forceAddCommentOpen === true)); }, [forceAddCommentOpen, isMobile]);
    const handleAddCommentOpen = useCallback(() => setIsAddCommentOpen(true), []);
    const handleAddCommentClose = useCallback(() => {
        setIsAddCommentOpen(false);
        if (onAddCommentClose) onAddCommentClose();
    }, [onAddCommentClose]);

    /**
     * When new comment is created, add it to the list of comments
     */
    const onCommentAdd = useCallback((comment: Comment) => {
        // Make comment first, so you can see it without having to scroll to the bottom
        setAllData(curr => [{
            __typename: "CommentThread",
            comment,
            childThreads: [],
            endCursor: null,
            totalInThread: 0,
        }, ...curr]);
        // Close add comment input
        if (isMobile) handleAddCommentClose();
    }, [handleAddCommentClose, isMobile, setAllData]);

    // The add component is always visible on desktop.
    // If forceAddCommentOpen is true (i.e. parent container wants add comment to be open), 
    // then we should scroll and focus the add comment input
    useEffect(() => {
        if (!forceAddCommentOpen || isMobile) return;
        scrollIntoFocusedView("markdown-input-add-comment-root");
    }, [forceAddCommentOpen, isMobile]);

    return (
        <ContentCollapse isOpen={isOpen} title={`Comments (${allData.length})`}>
            <CommentUpsert
                display="dialog"
                isCreate={true}
                isOpen={isAddCommentOpen}
                language={language}
                objectId={objectId}
                objectType={objectType}
                onCancel={handleAddCommentClose}
                onClose={handleAddCommentClose}
                onCompleted={onCommentAdd}
                onDeleted={handleAddCommentClose}
                parent={null} // parent is the thread. This is a top-level comment, so no parent
            />
            <Stack direction="column" spacing={2}>
                {/* Add comment button */}
                {!isAddCommentOpen && isMobile && <Button
                    fullWidth
                    startIcon={<CreateIcon />}
                    onClick={handleAddCommentOpen}
                    sx={{ marginTop: 2 }}
                    variant="outlined"
                >{t("AddComment")}</Button>}
                {/* Sort & filter */}
                {allData.length > 0 && <>
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
                    {/* Comments list */}
                    <Stack direction="column" spacing={2}>
                        {allData.map((thread, index) => (
                            <CommentThread
                                key={index}
                                canOpen={true}
                                data={thread}
                                language={language}
                            />
                        ))}
                    </Stack>
                </>}
            </Stack>
        </ContentCollapse>
    );
}
