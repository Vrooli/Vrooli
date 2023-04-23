import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { noteVersionCreate } from "../../../api/generated/endpoints/noteVersion_create";
import { noteVersionFindOne } from "../../../api/generated/endpoints/noteVersion_findOne";
import { noteVersionUpdate } from "../../../api/generated/endpoints/noteVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { NoteForm, noteInitialValues, transformNoteValues, validateNoteValues } from "../../../forms/NoteForm/NoteForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const NoteUpsert = ({ display = "page", isCreate, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(noteVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => noteInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(noteVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(noteVersionUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateNote" : "UpdateNote",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformNoteValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateNoteValues(values, existing), children: (formik) => _jsx(NoteForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, versions: [], zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=NoteUpsert.js.map