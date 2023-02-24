/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Button, Stack, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { useLazyQuery } from 'api/hooks';
import { CommentCreateInput } from 'components/inputs';
import { getUserLanguages, searchTypeToParams, useWindowSize } from 'utils';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { Wrap } from 'types';
import { CommentThread } from 'components/lists/comment';
import { uuidValidate } from '@shared/uuid';
import { SearchButtonsList } from 'components/lists';
import { CreateIcon } from '@shared/icons';
import { ContentCollapse } from '../ContentCollapse/ContentCollapse';
import { CommentThread as ThreadType, CommentSearchInput, CommentSearchResult, CommentSortBy, TimeFrame, Comment } from '@shared/consts';
import { commentFindMany } from 'api/generated/endpoints/comment';
import { SearchParams } from 'utils/search/schemas/base';
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
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const [{ advancedSearchSchema, defaultSortBy, sortByOptions }, setSearchParams] = useState<Partial<SearchParams>>({});
    useEffect(() => {
        async function getSearchParams() {
            setSearchParams(await searchTypeToParams.Comment(getUserLanguages(session)[0]));
        }
        getSearchParams();
    }, [session]);

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
    }, [defaultSortBy, sortByOptions]);

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

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object | null>({});
    const [getPageData, { data: pageData, loading }] = useLazyQuery<CommentSearchResult, CommentSearchInput, 'comments'>(commentFindMany, 'comments', {
        variables: {
            after: after.current,
            take: 20,
            sortBy: sortBy as CommentSortBy,
            searchString,
            createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
                after: timeFrame.after?.toISOString(),
                before: timeFrame.before?.toISOString(),
            } : undefined,
            [`${objectType.toLowerCase()}Id`]: objectId,
            ...advancedSearchParams
        },
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
    const parseData = useCallback((data: Wrap<CommentSearchResult, 'comments'> | undefined): ThreadType[] => {
        if (!data) return [];
        return data.comments.threads ?? [];
    }, []);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
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
    }, [pageData, parseData]);

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