/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Button, Stack, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { CommentCreateInput } from 'components/inputs';
import { getUserLanguages, useFindMany, useWindowSize } from 'utils';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { CommentThread } from 'components/lists/comment';
import { uuidValidate } from '@shared/uuid';
import { SearchButtonsList } from 'components/lists';
import { CreateIcon } from '@shared/icons';
import { ContentCollapse } from '../ContentCollapse/ContentCollapse';
import { CommentThread as ThreadType, Comment } from '@shared/consts';
import { useTranslation } from 'react-i18next';

export function CommentContainer({
    forceAddCommentOpen,
    isOpen,
    language,
    objectId,
    objectType,
    onAddCommentClose,
    session,
    zIndex,
}: CommentContainerProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

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
        canSearch: uuidValidate(objectId),
        searchType: 'Comment',
        resolve: (result) => result.threads,
        session,
        where: {
            [`${objectType.toLowerCase()}Id`]: objectId,
        },
    });

    /**
     * When new comment is created, add it to the list of comments
     */
    const onCommentAdd = useCallback((comment: Comment) => {
        // Make comment first, so you can see it without having to scroll to the bottom
        setAllData(curr => [{
            __typename: 'CommentThread',
            comment: comment as any,
            childThreads: [],
            endCursor: null,
            totalInThread: 0,
        }, ...curr]);
    }, [setAllData]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(isMobile);
    // Show add comment input if on desktop. For mobile, we'll show a button
    useEffect(() => { setIsAddCommentOpen(!isMobile || (forceAddCommentOpen === true)) }, [forceAddCommentOpen, isMobile]);
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
        const addCommentInput = document.getElementById('markdown-input-add-comment-root');
        if (addCommentInput) {
            addCommentInput.scrollIntoView({ behavior: 'smooth' });
            addCommentInput.focus();
        }
    }, [forceAddCommentOpen, isMobile]);

    return (
        <ContentCollapse isOpen={isOpen} session={session} title="Comments">
            {/* Add comment */}
            {
                isAddCommentOpen && <CommentCreateInput
                    handleClose={handleAddCommentClose}
                    language={language}
                    objectId={objectId}
                    objectType={objectType}
                    onCommentAdd={onCommentAdd}
                    parent={null} // parent is the thread. This is a top-level comment, so no parent
                    session={session}
                    zIndex={zIndex}
                />
            }
            {/* Sort & filter */}
            {allData.length > 0 ? <>
                <SearchButtonsList
                    advancedSearchParams={advancedSearchParams}
                    advancedSearchSchema={advancedSearchSchema}
                    searchType="Comment"
                    session={session}
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
                            session={session}
                            zIndex={zIndex}
                        />
                    ))}
                </Stack>
            </> : (!isAddCommentOpen && isMobile) ? <Button
                fullWidth
                startIcon={<CreateIcon />}
                onClick={handleAddCommentOpen}
                sx={{ marginTop: 2 }}
            >{t(`common:AddComment`, { lng })}</Button> : null}
        </ContentCollapse>
    );
}