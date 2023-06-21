import { BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, endpointGetBookmarkList, endpointPostBookmarkList, endpointPutBookmarkList, FindByIdInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BookmarkListForm, bookmarkListInitialValues, transformBookmarkListValues, validateBookmarkListValues } from "forms/BookmarkListForm/BookmarkListForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { BookmarkListUpsertProps } from "../types";

export const BookmarkListUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: BookmarkListUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl({}), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, BookmarkList>(endpointGetBookmarkList);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => bookmarkListInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<BookmarkList>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<BookmarkListCreateInput, BookmarkList>(endpointPostBookmarkList);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<BookmarkListUpdateInput, BookmarkList>(endpointPutBookmarkList);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateBookmarkList" : "UpdateBookmarkList")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>({
                        fetch,
                        inputs: transformBookmarkListValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateBookmarkListValues(values, existing)}
            >
                {(formik) => <BookmarkListForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
