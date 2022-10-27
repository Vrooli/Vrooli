/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Box, Button, Palette, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { useLazyQuery } from '@apollo/client';
import { CommentCreateInput } from 'components/inputs';
import { addSearchParams, labelledSortOptions, parseSearchParams, removeSearchParams, SearchType, searchTypeToParams, SortValueToLabelMap, useWindowSize } from 'utils';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { TimeFrame } from 'graphql/generated/globalTypes';
import { comments, commentsVariables } from 'graphql/generated/comments';
import { useLocation } from '@shared/route';
import { commentsQuery } from 'graphql/query';
import { Comment, CommentThread as ThreadType } from 'types';
import { CommentThread } from 'components/lists/comment';
import { uuidValidate } from '@shared/uuid';
import { AdvancedSearchDialog } from 'components/dialogs';
import { SortMenu, TimeMenu } from 'components/lists';
import { BuildIcon, SortIcon, HistoryIcon as TimeIcon, CreateIcon } from '@shared/icons';
import { CommentUpdateInput } from 'components/inputs/CommentUpdateInput/CommentUpdateInput';
import { ContentCollapse } from '../ContentCollapse/ContentCollapse';

const { advancedSearchSchema, defaultSortBy, sortByOptions } = searchTypeToParams.Comment;
const sortOptionsLabelled = labelledSortOptions(sortByOptions);

const searchButtonStyle = (palette: Palette) => ({
    minHeight: '34px',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    borderRadius: '50px',
    border: `2px solid ${palette.secondary.main}`,
    margin: 1,
    padding: 0,
    paddingLeft: 1,
    paddingRight: 1,
    cursor: 'pointer',
    '&:hover': {
        transform: 'scale(1.1)',
    },
    transition: 'transform 0.2s ease-in-out',
});

export function CommentContainer({
    language,
    objectId,
    objectType,
    session,
    zIndex,
}: CommentContainerProps) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const [, setLocation] = useLocation();

    const [sortBy, setSortBy] = useState<string>(defaultSortBy);
    const [searchString, setSearchString] = useState<string>('');
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
        if (typeof searchParams.sort === 'string') {
            // Check if sortBy is valid
            if (searchParams.sort in sortByOptions) {
                setSortBy(searchParams.sort);
            } else {
                setSortBy(defaultSortBy);
            }
        }
        if (typeof searchParams.time === 'object' &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty('after') &&
            searchParams.time.hasOwnProperty('before')) {
            setTimeFrame({
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            });
        }
    }, []);

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('');
    const after = useRef<string | undefined>(undefined);

    /**
     * When sort and filter options change, update the URL
     */
    useEffect(() => {
        addSearchParams(setLocation, {
            search: searchString.length > 0 ? searchString : undefined,
            sort: sortBy,
            time: timeFrame ? {
                after: timeFrame.after?.toISOString() ?? '',
                before: timeFrame.before?.toISOString() ?? '',
            } : undefined,
        });
    }, [searchString, sortBy, timeFrame, setLocation]);

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object>({});
    const [getPageData, { data: pageData, loading }] = useLazyQuery<comments, commentsVariables>(commentsQuery, {
        variables: ({
            input: {
                after: after.current,
                take: 20,
                sortBy,
                searchString,
                createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
                    after: timeFrame.after?.toISOString(),
                    before: timeFrame.before?.toISOString(),
                } : undefined,
                [`${objectType.toLowerCase()}Id`]: objectId,
                ...advancedSearchParams
            }
        } as any),
        errorPolicy: 'all',
    });
    const [allData, setAllData] = useState<ThreadType[]>([]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        after.current = undefined;
        if (uuidValidate(objectId)) getPageData();
    }, [advancedSearchParams, searchString, sortBy, timeFrame, getPageData, objectId]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!pageData || !uuidValidate(objectId)) return;
        const queryData: any = Object.values(pageData)[0];
        if (!queryData || !queryData.pageInfo) return [];
        if (queryData.pageInfo?.hasNextPage) {
            const { endCursor } = queryData.pageInfo;
            if (endCursor) {
                after.current = endCursor;
                getPageData();
            }
        }
    }, [getPageData, objectId, pageData]);

    /**
     * Helper method for converting fetched data to an array of object data
     */
    const parseData = useCallback((data: comments | undefined): ThreadType[] => {
        if (!data) return [];
        return data.comments.threads ?? [];
    }, []);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, frame?: { after?: Date | undefined, before?: Date | undefined }) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (label) setTimeFrameLabel(label === 'All Time' ? '' : label);
    };

    /**
     * Find sort by label when sortBy changes
     */
    const sortByLabel = useMemo(() => {
        if (sortBy && sortBy in SortValueToLabelMap) return SortValueToLabelMap[sortBy];
        return '';
    }, [sortBy]);

    // Handle advanced search
    useEffect(() => {
        const searchParams = parseSearchParams();
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === 'boolean') setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema?.fields?.map(f => f.fieldName) ?? [];
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, []);

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true) }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false)
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        // Remove undefined and 0 values
        const valuesWithoutBlanks = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== undefined && v !== 0));
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.fields?.map(f => f.fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, valuesWithoutBlanks);
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [setLocation]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        // Close advanced search dialog
        // handleAdvancedSearchDialogClose();
        const parsedData = parseData(pageData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (after.current) {
            setAllData(curr => [...curr, ...parsedData]);
        } else {
            setAllData(parsedData);
        }
    }, [pageData, parseData, handleAdvancedSearchDialogClose]);

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
    }, []);

    const onCommentUpdate = useCallback((comment: Comment) => {
        setAllData(curr => {
            const index = curr.findIndex(c => c.comment.id === comment.id);
            if (index === -1) return curr;
            const newCurr = [...curr];
            newCurr[index] = {
                ...newCurr[index],
                comment,
            };
            return newCurr;
        });
    }, []);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(isMobile);
    // Show add comment input if on desktop. For mobile, we'll show a button
    useEffect(() => { setIsAddCommentOpen(!isMobile) }, [isMobile]);
    const [commentToUpdate, setCommentToUpdate] = useState<Comment | null>(null);
    const handleAddCommentOpen = useCallback(() => setIsAddCommentOpen(true), []);
    const handleAddCommentClose = useCallback(() => setIsAddCommentOpen(false), []);
    const handleUpdateCommentOpen = useCallback((comment: Comment) => { setCommentToUpdate(comment) }, []);
    const handleUpdateCommentClose = useCallback(() => { setCommentToUpdate(null) }, []);

    return (
        <ContentCollapse title="Comments">
            {/* Dialog for setting advanced search items */}
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={handleAdvancedSearchDialogSubmit}
                isOpen={advancedSearchDialogOpen}
                searchType={SearchType.Comment}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Menu for selecting "sort by" type */}
            <SortMenu
                sortOptions={sortOptionsLabelled}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            {/* Menu for selecting time created */}
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
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
            {/* Update comment */}
            {
                commentToUpdate && <CommentUpdateInput
                    comment={commentToUpdate}
                    handleClose={handleUpdateCommentClose}
                    language={language}
                    objectId={objectId}
                    objectType={objectType}
                    onCommentUpdate={onCommentUpdate}
                    parent={null} // parent is the thread. This is a top-level comment, so no parent
                    session={session}
                    zIndex={zIndex}
                />
            }
            {/* Sort & filter */}
            {allData.length > 0 ? <>
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
                    <Tooltip title="Sort by" placement="top">
                        <Box
                            onClick={handleSortOpen}
                            sx={searchButtonStyle(palette)}
                        >
                            <SortIcon fill={palette.secondary.main} />
                            <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{sortByLabel}</Typography>
                        </Box>
                    </Tooltip>
                    <Tooltip title="Time created" placement="top">
                        <Box
                            onClick={handleTimeOpen}
                            sx={searchButtonStyle(palette)}
                        >
                            <TimeIcon fill={palette.secondary.main} />
                            <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{timeFrameLabel}</Typography>
                        </Box>
                    </Tooltip>
                    {advancedSearchParams && <Tooltip title="See all search settings" placement="top">
                        <Box
                            onClick={handleAdvancedSearchDialogOpen}
                            sx={searchButtonStyle(palette)}
                        >
                            <BuildIcon fill={palette.secondary.main} />
                            {Object.keys(advancedSearchParams).length > 0 && <Typography variant="body2" sx={{ marginLeft: 0.5 }}>
                                *{Object.keys(advancedSearchParams).length}
                            </Typography>}
                        </Box>
                    </Tooltip>}
                </Box>
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
            >Add comment</Button> : null}
        </ContentCollapse>
    );
}