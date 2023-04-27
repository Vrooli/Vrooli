import { BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, FindByIdInput } from "@local/shared";
import { bookmarkListCreate } from "api/generated/endpoints/bookmarkList_create";
import { bookmarkListFindOne } from "api/generated/endpoints/bookmarkList_findOne";
import { bookmarkListUpdate } from "api/generated/endpoints/bookmarkList_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BookmarkListForm, bookmarkListInitialValues, transformBookmarkListValues, validateBookmarkListValues } from "forms/BookmarkListForm/BookmarkListForm";
import { useContext, useEffect, useMemo, useRef } from "react";
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
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<BookmarkList, FindByIdInput>(bookmarkListFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => bookmarkListInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<BookmarkList>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<BookmarkList, BookmarkListCreateInput>(bookmarkListCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<BookmarkList, BookmarkListUpdateInput>(bookmarkListUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateBookmarkList" : "UpdateBookmarkList",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<BookmarkList, BookmarkListCreateInput | BookmarkListUpdateInput>({
                        mutation,
                        input: transformBookmarkListValues(values, existing),
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
