import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useMutation } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { ROLES, routineCreateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineCreateMutation } from "graphql/mutation";
import { formatForCreate, getUserLanguages, updateArray, useReactSearch } from "utils";
import { RoutineCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { LanguageInput, MarkdownInput, ResourceListHorizontal, TagSelector, UserOrganizationSwitch } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { NewObject, Organization, ResourceList, Routine, RoutineInputList, RoutineOutputList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuidv4 } from 'uuid';
import { InputOutputContainer } from "components/lists/inputOutput";

export const RoutineCreate = ({
    onCreated,
    onCancel,
    session,
}: RoutineCreateProps) => {
    const params = useReactSearch(null);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, [setOrganizationFor]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<RoutineInputList>([]);
    const handleInputsUpdate = useCallback((updatedList: RoutineInputList) => {
        console.log('HANDLE INPUTS UPDATEEEEEE', updatedList)
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<RoutineOutputList>([]);
    const handleOutputsUpdate = useCallback((updatedList: RoutineOutputList) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle translations
    type Translation = NewObject<Routine['translations'][0]>;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        console.log('IN DELETE TRANS A', translations)
        console.log('IN DELETE TRANS B', [...translations.filter(t => t.language !== language)])
        setTranslations([...translations.filter(t => t.language !== language)]);
        // Also delete translations from inputs and outputs
        setInputsList(inputsList.map(i => {
            const updatedTranslationsList = i.translations.filter(t => t.language !== language);
            return { ...i, translations: updatedTranslationsList };
        }));
        setOutputsList(outputsList.map(o => {
            const updatedTranslationsList = o.translations.filter(t => t.language !== language);
            return { ...o, translations: updatedTranslationsList };
        }));
    }, [translations, inputsList, outputsList]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        console.log('updateTranslation', language, translation)
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    useEffect(() => {
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, [params]);

    // Handle create
    const [mutation] = useMutation<routine>(routineCreateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            instructions: 'Fill out the form below.',
            title: '',
            version: '1.0',
            isComplete: true,
        },
        validationSchema,
        onSubmit: (values) => {
            const resourceListAdd = resourceList ? formatForCreate(resourceList) : {};
            const tagsAdd = tags.length > 0 ? {
                tagsCreate: tags.filter(t => !t.id).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id).map(t => (t.id)),
            } : {};
            const createdFor = organizationFor ? { createdByOrganizationId: organizationFor.id } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            mutationWrapper({
                mutation,
                input: formatForCreate({
                    ...createdFor,
                    translations: allTranslations,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceListsCreate: [resourceListAdd],
                    ...tagsAdd,
                    version: values.version,
                    isComplete: values.isComplete,
                }) as any,
                onSuccess: (response) => { onCreated(response.data.routineCreate) },
                onError: () => { formik.setSubmitting(false) }
            })
        },
    });

    // Handle languages
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [languages, setLanguages] = useState<string[]>([getUserLanguages(session)[0]]);
    useEffect(() => {
        if (languages.length === 0) {
            const userLanguage = getUserLanguages(session)[0]
            setLanguage(userLanguage)
            setLanguages([userLanguage])
        }
    }, [languages, session, setLanguage, setLanguages])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            instructions: existingTranslation?.instructions ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        console.log('handleLanguageSelect', newLanguage);
        // Update old select
        updateTranslation(language, {
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.instructions, formik.values.title, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        console.log('HANDLE LANGUAGE DELETE', language)
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        deleteTranslation(language);
        updateFormikTranslation(newLanguages[0]);
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
    }, [deleteTranslation, languages, updateFormikTranslation]);

    const actions: DialogActionItem[] = useMemo(() => {
        const correctRole = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        return [
            ['Create', CreateIcon, Boolean(!correctRole || formik.isSubmitting), true, () => { }],
            ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
        ] as DialogActionItem[]
    }, [formik, onCancel, session]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            paddingBottom: `${formBottom}px`,
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <UserOrganizationSwitch session={session} selected={organizationFor} onChange={onSwitchChange} />
                </Grid>
                {/* TODO add project selector */}
                <Grid item xs={12}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleLanguageDelete}
                        handleCurrent={handleLanguageSelect}
                        selectedLanguages={languages}
                        session={session}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="title"
                        name="title"
                        label="title"
                        value={formik.values.title}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.title && Boolean(formik.errors.title)}
                        helperText={formik.touched.title && formik.errors.title}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        value={formik.values.description}
                        rows={3}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={formik.touched.description && formik.errors.description}
                    />
                </Grid>
                <Grid item xs={12}>
                    <MarkdownInput
                        id="instructions"
                        placeholder="Instructions"
                        value={formik.values.instructions}
                        minRows={4}
                        onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                        error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                        helperText={formik.touched.instructions ? formik.errors.instructions : null}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="version"
                        name="version"
                        label="version"
                        value={formik.values.version}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.version && Boolean(formik.errors.version)}
                        helperText={formik.touched.version && formik.errors.version}
                    />
                </Grid>
                <Grid item xs={12}>
                    <InputOutputContainer
                        isEditing={true}
                        handleUpdate={handleInputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                        isInput={true}
                        language={language}
                        list={inputsList}
                        session={session}
                    />
                </Grid>
                <Grid item xs={12}>
                    <InputOutputContainer
                        isEditing={true}
                        handleUpdate={handleOutputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                        isInput={false}
                        language={language}
                        list={outputsList}
                        session={session}
                    />
                </Grid>
                <Grid item xs={12}>
                    <ResourceListHorizontal
                        title={'Resources'}
                        list={resourceList}
                        canEdit={true}
                        handleUpdate={handleResourcesUpdate}
                        loading={false}
                        session={session}
                        mutate={false}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TagSelector
                        session={session}
                        tags={tags}
                        onTagAdd={addTag}
                        onTagRemove={removeTag}
                        onTagsClear={clearTags}
                    />
                </Grid>
                <Grid item xs={12} marginBottom={4}>
                    <Tooltip placement={'top'} title='Is this routine ready for anyone to use?'>
                        <FormControlLabel
                            label='Complete'
                            control={
                                <Checkbox
                                    id='routine-is-complete'
                                    name='isComplete'
                                    color='secondary'
                                    checked={formik.values.isComplete}
                                    onChange={formik.handleChange}
                                    onBlur={formik.handleBlur}
                                />
                            }
                        />
                    </Tooltip>
                </Grid>
            </Grid>
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}