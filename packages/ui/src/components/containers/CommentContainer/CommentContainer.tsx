/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Box, Grid, Stack, Typography, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { commentCreate as validationSchema, commentTranslationCreate } from '@shared/validation';
import { useLazyQuery, useMutation } from '@apollo/client';
import { commentCreate, commentCreateVariables } from 'graphql/generated/commentCreate';
import { MarkdownInput } from 'components/inputs';
import { useFormik } from 'formik';
import { commentCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils';
import { addSearchParams, DUMMY_ID, getFormikErrorsWithTranslations, getTranslationData, handleTranslationBlur, handleTranslationChange, PubSub, removeSearchParams, searchTypeToParams, usePromptBeforeUnload, useReactSearch } from 'utils';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { TimeFrame } from 'graphql/generated/globalTypes';
import { comments, commentsVariables } from 'graphql/generated/comments';
import { useLocation } from '@shared/route';
import { commentsQuery } from 'graphql/query';
import { Comment, CommentThread as ThreadType } from 'types';
import { CommentThread } from 'components/lists/comment';
import { validate as uuidValidate } from 'uuid';
import { v4 as uuid } from 'uuid';
import { GridSubmitButtons } from 'components/buttons';

const { advancedSearchSchema, defaultSortBy } = searchTypeToParams.Comment;

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

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('Time');
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
        })
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

    // Handle advanced search
    useEffect(() => {
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === 'boolean') setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema?.fields?.map(f => f.fieldName) ?? [];
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [searchParams]);

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

    const [addMutation, { loading: loadingAdd }] = useMutation<commentCreate, commentCreateVariables>(commentCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            createdFor: objectType,
            forId: objectId,
            translationsCreate: [{
                id: DUMMY_ID,
                language,
                text: '',
            }],
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: addMutation,
                input: {
                    id: uuid(),
                    createdFor: values.createdFor,
                    forId: values.forId,
                    translationsCreate: values.translationsCreate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
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
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current text, as well as errors
    const { text, errorText, touchedText, errors } = useMemo(() => {
        console.log('comment container gettransdata')
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            text: value?.text ?? '',
            errorText: error?.text ?? '',
            touchedText: touched?.text ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', commentTranslationCreate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

    const isLoggedIn = useMemo(() => session?.isLoggedIn === true && uuidValidate(session?.id ?? ''), [session]);

    return (
        <Box
            id="comments"
            sx={{
                overflow: 'overlay',
                background: palette.background.paper,
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
                        value={text}
                        minRows={3}
                        onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                        error={touchedText && Boolean(errorText)}
                        helperText={touchedText ? errorText : null}
                    />
                    <Grid container spacing={1} sx={{
                        width: 'min(100%, 400px)',
                        marginLeft: 'auto',
                        marginTop: 1,
                    }}>
                        <GridSubmitButtons
                            disabledSubmit={!isLoggedIn}
                            errors={errors}
                            isCreate={true}
                            loading={formik.isSubmitting || loadingAdd}
                            onCancel={formik.resetForm}
                            onSetSubmitting={formik.setSubmitting}
                            onSubmit={formik.submitForm}
                        />
                    </Grid>
                </Box>
            </form>
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
        </Box>
    );
}