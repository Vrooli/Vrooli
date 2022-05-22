import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { standard } from "graphql/generated/standard";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { InputType, ROLES, standardCreateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { standardCreateMutation } from "graphql/mutation";
import { formatForCreate, getUserLanguages, updateArray, useReactSearch } from "utils";
import { StandardCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { JsonInput, LanguageInput, ResourceListHorizontal, Selector, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { NewObject, ResourceList, Standard } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuidv4 } from 'uuid';
import { FieldData } from "forms/types";
import { createDefaultFieldData } from "forms/generators";

/**
 * Supported input types
 */
export const InputTypeOptions: { label: string, value: InputType }[] = [
    {
        label: 'Text',
        value: InputType.TextField,
    },
    {
        label: 'JSON',
        value: InputType.JSON,
    },
    {
        label: 'Integer',
        value: InputType.QuantityBox
    },
    {
        label: 'Radio (Select One)',
        value: InputType.Radio,
    },
    {
        label: 'Checkbox (Select any)',
        value: InputType.Checkbox,
    },
    {
        label: 'Switch (On/Off)',
        value: InputType.Switch,
    },
    {
        label: 'File Upload',
        value: InputType.Dropzone,
    },
    {
        label: 'Markdown',
        value: InputType.Markdown
    },
]

export const StandardCreate = ({
    onCreated,
    onCancel,
    session,
}: StandardCreateProps) => {
    const params = useReactSearch(null);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputType>(InputType.JSON);
    const handleInputTypeSelect = useCallback((e) => setInputType(e.target.value), []);

    // Handle standard schema
    const [schema, setSchema] = useState<FieldData>(createDefaultFieldData(inputType));

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
    type Translation = NewObject<Standard['translations'][0]>;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations, setTranslations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    useEffect(() => {
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, [params]);

    // Handle create
    const [mutation] = useMutation<standard>(standardCreateMutation);
    const formik = useFormik({
        initialValues: {
            default: '',
            description: '',
            name: '',
            schema: '',
            type: '',
            version: '',
        },
        validationSchema,
        onSubmit: (values) => {
            const resourceListAdd = resourceList ? formatForCreate(resourceList) : {};
            const tagsAdd = tags.length > 0 ? {
                tagsCreate: tags.filter(t => !t.id).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id).map(t => (t.id)),
            } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
            })
            mutationWrapper({
                mutation,
                input: formatForCreate({
                    default: values.default,
                    description: values.description,
                    name: values.name,
                    schema: values.schema,
                    translations: allTranslations,
                    resourceListsCreate: [resourceListAdd],
                    ...tagsAdd,
                    type: values.type,
                    version: values.version,
                }) as any,
                onSuccess: (response) => { onCreated(response.data.standardCreate) },
                onError: () => { formik.setSubmitting(false) },
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
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            description: formik.values.description,
        });
        // Change selection
        setLanguage(newLanguage);
        // Update languages
        const newLanguages = [...languages];
        const index = newLanguages.findIndex(l => l === oldLanguage);
        if (index >= 0) {
            newLanguages[index] = newLanguage;
            setLanguages(newLanguages);
        }
    }, [formik.values, languages, setLanguage, setLanguages, updateTranslation]);
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            language,
            description: formik.values.description,
        })
        // Update formik
        updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
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
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleChange={handleLanguageChange}
                        handleDelete={handleLanguageDelete}
                        handleSelect={handleLanguageSelect}
                        languages={languages}
                        session={session}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="name"
                        name="name"
                        label="Name"
                        value={formik.values.name}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.name && Boolean(formik.errors.name)}
                        helperText={formik.touched.name && formik.errors.name}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        multiline
                        minRows={4}
                        value={formik.values.description}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={formik.touched.description && formik.errors.description}
                    />
                </Grid>
                {/* Select the standard type */}
                <Grid item xs={12}>
                    <Selector
                        fullWidth
                        options={InputTypeOptions}
                        selected={inputType}
                        handleChange={handleInputTypeSelect}
                        inputAriaLabel='input-type-selector'
                        label="Size"
                    />
                </Grid>
                {/* Define the standard */}
                {/* <Grid item xs={12}>
                    <JsonInput
                        id="schema"
                        format={props.format}
                        variables={props.variables}
                        placeholder={props.placeholder ?? data.label}
                        value={formik.values[data.fieldName]}
                        minRows={props.minRows}
                        onChange={(newText: string) => formik.setFieldValue(data.fieldName, newText)}
                        error={formik.touched[data.fieldName] && Boolean(formik.errors[data.fieldName])}
                        helperText={formik.touched[data.fieldName] && formik.errors[data.fieldName]}
                    />
                </Grid> */}
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
                <Grid item xs={12} marginBottom={4}>
                    <TagSelector
                        session={session}
                        tags={tags}
                        onTagAdd={addTag}
                        onTagRemove={removeTag}
                        onTagsClear={clearTags}
                    />
                </Grid>
            </Grid>
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}