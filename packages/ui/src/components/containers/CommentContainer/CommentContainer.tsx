/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Box, Button, CircularProgress, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { containerShadow } from 'styles';
import { commentCreateForm as validationSchema } from '@local/shared';
import { useLazyQuery, useMutation } from '@apollo/client';
import { commentCreate, commentCreateVariables } from 'graphql/generated/commentCreate';
import { MarkdownInput } from 'components/inputs';
import { useFormik } from 'formik';
import { commentCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils';
import { objectToSearchInfo, ObjectType, parseSearchParams, PubSub, stringifySearchParams, useReactSearch } from 'utils';
import { useCallback, useEffect, useRef, useState } from 'react';
import { TimeFrame } from 'graphql/generated/globalTypes';
import { comments, commentsVariables } from 'graphql/generated/comments';
import { useLocation } from 'wouter';
import { commentsQuery } from 'graphql/query';
import { Comment, CommentThread as ThreadType } from 'types';
import { CommentThread } from 'components/lists/comment';
import { validate as uuidValidate } from 'uuid';
import { v4 as uuid } from 'uuid';

const { advancedSearchSchema, defaultSortBy, sortByOptions } = objectToSearchInfo[ObjectType.Comment];

export function CommentContainer({
    language,
    objectId,
    objectType,
    session,
    sxs,
    zIndex,
}: CommentContainerProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const [sortBy, setSortBy] = useState<string>(defaultSortBy);
    const [searchString, setSearchString] = useState<string>('');
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    // Handle URL params
    const searchParams = useReactSearch(null);
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
        if (typeof searchParams.sort === 'string') setSortBy(searchParams.sort);
        if (typeof searchParams.time === 'object' &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty('after') &&
            searchParams.time.hasOwnProperty('before')) {
            setTimeFrame({
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            });
        }
    }, [searchParams]);

    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('Time');
    const after = useRef<string | undefined>(undefined);

    /**
     * When sort and filter options change, update the URL
     */
    useEffect(() => {
        const params = parseSearchParams(window.location.search);
        if (searchString) params.search = searchString;
        else delete params.search;
        if (sortBy) params.sort = sortBy;
        else delete params.sort;
        if (timeFrame) {
            params.time = {
                after: timeFrame.after?.toISOString() ?? '',
                before: timeFrame.before?.toISOString() ?? '',
            }
        }
        else delete params.time;
        setLocation(stringifySearchParams(params), { replace: true });
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
        } as any)
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

    // Handle advanced search
    useEffect(() => {
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === 'boolean') setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.fields.map(f => f.fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema.fields, searchParams]);

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true) }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false)
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        // Remove undefined and 0 values
        const valuesWithoutBlanks = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== undefined && v !== 0));
        // Add advanced search params to url search params
        setLocation(stringifySearchParams({
            ...searchParams,
            ...valuesWithoutBlanks
        }));
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [searchParams, setLocation]);

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

    const [addMutation, { loading: loadingAdd }] = useMutation<commentCreate, commentCreateVariables>(commentCreateMutation);
    const formik = useFormik({
        initialValues: {
            comment: '',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: addMutation,
                input: {
                    id: uuid(),
                    createdFor: objectType,
                    forId: objectId,
                    translationsCreate: [{
                        id: uuid(),
                        language,
                        text: values.comment,
                    }]
                },
                successCondition: (response) => response.data.commentCreate !== null,
                onSuccess: (response) => {
                    PubSub.get().publishSnack({ message: 'Comment created.', severity: 'success' });
                    formik.resetForm();
                    onCommentAdd(response.data.commentCreate);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    /**
     * Handle add comment click
     */
    const handleAddComment = useCallback((event: any) => {
        // Make sure submit does not propagate past the form
        event.preventDefault();
        // Make sure form is valid
        if (!formik.isValid) return;
        // Submit form
        formik.submitForm();
    }, [formik]);

    return (
        <Box
            id="comments"
            sx={{
                ...containerShadow,
                borderRadius: '8px',
                overflow: 'overlay',
                background: palette.background.default,
                width: 'min(100%, 700px)',
                ...(sxs?.root ?? {}),
            }}
        >
            {/* Add comment */}
            <form>
                <Box sx={{ margin: 2 }}>
                    <Typography component="h3" variant="h6" textAlign="left">Add comment</Typography>
                    <MarkdownInput
                        id="add-comment"
                        placeholder="Please be nice to each other."
                        value={formik.values.comment}
                        minRows={3}
                        onChange={(newText: string) => formik.setFieldValue('comment', newText)}
                        error={formik.touched.comment && Boolean(formik.errors.comment)}
                        helperText={formik.touched.comment ? formik.errors.comment as string : null}
                    />
                    <Box sx={{
                        marginTop: 1,
                        display: 'flex',
                        flexDirection: 'row-reverse',
                    }}>
                        <Tooltip title={formik.errors.comment ? formik.errors.comment as string : ''}>
                            <Button
                                color="secondary"
                                disabled={loadingAdd || formik.isSubmitting || !formik.isValid}
                                onClick={handleAddComment}
                            >
                                {loadingAdd ? <CircularProgress size={24} /> : 'Add'}
                            </Button>
                        </Tooltip>
                    </Box>
                </Box>
            </form>
            {/* Comments list */}
            <Stack direction="column" spacing={2}>
                {allData.map((thread, index) => (
                    <CommentThread
                        key={index}
                        data={thread}
                        language={language}
                        session={session}
                        zIndex={zIndex}
                    />
                ))}
            </Stack>
        </Box>
    );
}