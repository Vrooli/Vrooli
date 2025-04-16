import { ApiShape, ApiVersion, ApiVersionCreateInput, ApiVersionShape, apiVersionTranslationValidation, ApiVersionUpdateInput, apiVersionValidation, CodeLanguage, DUMMY_ID, endpointsApiVersion, LINKS, LlmTask, noopSubmit, orDefault, SearchPageTabOption, Session, shapeApiVersion } from "@local/shared";
import { Box, Button, Divider, Grid, InputAdornment, Typography } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures, summaryInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { PageContainer } from "../../../components/Page/Page.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, ScrollBox } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ApiFormProps, ApiUpsertProps } from "./types.js";

function apiInitialValues(
    session: Session | undefined,
    existing?: Partial<ApiVersion> | null | undefined,
): ApiVersionShape {
    return {
        __typename: "ApiVersion" as const,
        id: DUMMY_ID,
        callLink: "",
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        resourceList: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "ApiVersion" as const,
                id: DUMMY_ID,
            },
        },
        schemaLanguage: CodeLanguage.Yaml,
        schemaText: "",
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Api" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "ApiVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            details: "",
            name: "",
            summary: "",
        }]),
    };
}

function transformApiVersionValues(values: ApiVersionShape, existing: ApiVersionShape, isCreate: boolean) {
    return isCreate ? shapeApiVersion.create(values) : shapeApiVersion.update(existing, values);
}

const exampleCodeJson = `{
  "openapi": "3.1.0",
  "info": {
    "title": "TODO API",
    "description": "A simple API to manage TODO tasks.",
    "version": "1.0.0",
    "contact": {
      "name": "API Support",
      "email": "support@example.com"
    },
    "license": {
      "name": "Apache 2.0",
      "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
    }
  },
  "servers": [
    {
      "url": "https://api.todoapp.com/v1",
      "description": "Main (production) server"
    },
    {
      "url": "https://staging-api.todoapp.com/v1",
      "description": "Staging server"
    }
  ],
  "paths": {
    "/tasks": {
      "get": {
        "summary": "List all tasks",
        "operationId": "listTasks",
        "tags": ["tasks"],
        "responses": {
          "200": {
            "description": "A list of tasks",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "$ref": "#/components/schemas/Task"
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "summary": "Create a new task",
        "operationId": "createTask",
        "tags": ["tasks"],
        "requestBody": {
          "description": "The task to create",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/NewTask"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Task created successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Task"
                }
              }
            }
          }
        }
      }
    },
    "/tasks/{taskId}": {
      "get": {
        "summary": "Get a specific task",
        "operationId": "getTaskById",
        "tags": ["tasks"],
        "parameters": [
          {
            "name": "taskId",
            "in": "path",
            "required": true,
            "description": "The id of the task to retrieve",
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "The requested task",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Task"
                }
              }
            }
          }
        }
      },
      "put": {
        "summary": "Update a specific task",
        "operationId": "updateTask",
        "tags": ["tasks"],
        "parameters": [
          {
            "name": "taskId",
            "in": "path",
            "required": true,
            "description": "The id of the task to update",
            "schema": {
              "type": "string"
            }
          }
        ],
        "requestBody": {
          "description": "The updated task data",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/NewTask"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Task updated successfully",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Task"
                }
              }
            }
          }
        }
      },
      "delete": {
        "summary": "Delete a specific task",
        "operationId": "deleteTask",
        "tags": ["tasks"],
        "parameters": [
          {
            "name": "taskId",
            "in": "path",
            "required": true,
            "description": "The id of the task to delete",
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "204": {
            "description": "Task deleted successfully"
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Task": {
        "type": "object",
        "required": ["id", "title"],
        "properties": {
          "id": {
            "type": "string",
            "example": "1"
          },
          "title": {
            "type": "string",
            "example": "Buy groceries"
          },
          "description": {
            "type": "string",
            "example": "Milk, Bread, Eggs"
          },
          "completed": {
            "type": "boolean",
            "example": false
          }
        }
      },
      "NewTask": {
        "type": "object",
        "required": ["title"],
        "properties": {
          "title": {
            "type": "string",
            "example": "Buy groceries"
          },
          "description": {
            "type": "string",
            "example": "Milk, Bread, Eggs"
          },
          "completed": {
            "type": "boolean",
            "example": false
          }
        }
      }
    }
  }
}`;

const exampleCodeYaml = `openapi: 3.1.0
info:
  title: TODO API
  description: A simple API to manage TODO tasks.
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com
  license:
    name: Apache 2.0
    url: 'http://www.apache.org/licenses/LICENSE-2.0.html'
servers:
  - url: 'https://api.todoapp.com/v1'
    description: Main (production) server
  - url: 'https://staging-api.todoapp.com/v1'
    description: Staging server
paths:
  /tasks:
    get:
      summary: List all tasks
      operationId: listTasks
      tags:
        - tasks
      responses:
        '200':
          description: A list of tasks
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Task'
    post:
      summary: Create a new task
      operationId: createTask
      tags:
        - tasks
      requestBody:
        description: The task to create
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewTask'
      responses:
        '201':
          description: Task created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
  /tasks/{taskId}:
    get:
      summary: Get a specific task
      operationId: getTaskById
      tags:
        - tasks
      parameters:
        - name: taskId
          in: path
          required: true
          description: The id of the task to retrieve
          schema:
            type: string
      responses:
        '200':
          description: The requested task
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
    put:
      summary: Update a specific task
      operationId: updateTask
      tags:
        - tasks
      parameters:
        - name: taskId
          in: path
          required: true
          description: The id of the task to update
          schema:
            type: string
      requestBody:
        description: The updated task data
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewTask'
      responses:
        '200':
          description: Task updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Task'
    delete:
      summary: Delete a specific task
      operationId: deleteTask
      tags:
        - tasks
      parameters:
        - name: taskId
          in: path
          required: true
          description: The id of the task to delete
          schema:
            type: string
      responses:
        '204':
          description: Task deleted successfully
components:
  schemas:
    Task:
      type: object
      required:
        - id
        - title
      properties:
        id:
          type: string
          example: '1'
        title:
          type: string
          example: 'Buy groceries'
        description:
          type: string
          example: 'Milk, Bread, Eggs'
        completed:
          type: boolean
          example: false
    NewTask:
      type: object
      required:
        - title
      properties:
        title:
          type: string
          example: 'Buy groceries'
        description:
          type: string
          example: 'Milk, Bread, Eggs'
        completed:
          type: boolean
          example: false
`;

const callLinkInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Link" />
        </InputAdornment>
    ),
} as const;
const codeLimitTo = [CodeLanguage.Json, CodeLanguage.Graphql, CodeLanguage.Yaml] as const;
const formSectionTitleStyle = { marginBottom: 1 } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const schemaCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;
const dividerStyle = { display: { xs: "flex", md: "none" } } as const;

function ApiForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    versions,
    ...props
}: ApiFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: apiVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const [hasDocUrl, setHasDocUrl] = useState(false);
    const toggleHasDocUrlTrue = useCallback(() => setHasDocUrl(!hasDocUrl), [hasDocUrl]);

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "ApiVersion", id: values.id } as const;
    }, [values]);

    const { handleCancel, handleCompleted } = useUpsertActions<ApiVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "ApiVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsApiVersion.createOne,
        endpointUpdate: endpointsApiVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "ApiVersion" });

    const onSubmit = useSubmitHelper<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformApiVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            callLink: values.callLink,
            documentationLink: values.documentationLink,
            isPrivate: values.isPrivate,
            schemaLanguage: values.schemaLanguage,
            schemaText: values.schemaText,
            version: values.versionLabel,
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["name", "summary", "details"]);
        delete rest.id;
        const callLink = typeof rest.callLink === "string" ? rest.callLink : values.callLink;
        const documentationLink = typeof rest.documentationLink === "string" ? rest.documentationLink : values.documentationLink;
        const isPrivate = typeof rest.isPrivate === "boolean" ? rest.isPrivate : values.isPrivate;
        const schemaLanguage = typeof rest.schemaLanguage === "string" ? rest.schemaLanguage : values.schemaLanguage;
        const schemaText = typeof rest.schemaText === "string" ? rest.schemaText : values.schemaText;
        const versionLabel = typeof rest.version === "string" ? rest.version : values.versionLabel;
        const updatedValues = {
            ...values,
            callLink,
            documentationLink,
            isPrivate,
            schemaLanguage,
            schemaText,
            translations: updatedTranslations,
            versionLabel,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.ApiAdd : LlmTask.ApiUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const [codeLanguage, , codeLanguageHelpers] = useField<CodeLanguage>("schemaLanguage");
    const [, , contentHelpers] = useField<string>("schemaText");
    function showExample() {
        setHasDocUrl(false);
        let language = codeLanguage.value;
        // We only have an example for JavaScript and Yaml, so if the current language is not one of those, we'll default to Yaml
        if (![CodeLanguage.Javascript, CodeLanguage.Yaml].includes(language)) {
            language = CodeLanguage.Yaml;
            codeLanguageHelpers.setValue(language);
        }
        // Set value to hard-coded example
        if (language === CodeLanguage.Javascript) {
            contentHelpers.setValue(exampleCodeJson);
        } else if (language === CodeLanguage.Yaml) {
            contentHelpers.setValue(exampleCodeYaml);
        }
    }

    return (
        <MaybeLargeDialog
            display={display}
            id="api-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateApi" : "UpdateApi")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Api}"`}
                text="Search existing apis"
            />
            <BaseForm
                display={display}
                maxWidth={1200}
                isLoading={isLoading}
            >
                <FormContainer>
                    <Grid container spacing={2}>
                        <Grid item xs={12} md={6}>
                            <Box width="100%" padding={2}>
                                <Typography variant="h4" sx={formSectionTitleStyle}>Basic info</Typography>
                                <Box display="flex" flexDirection="column" gap={4}>
                                    <RelationshipList
                                        isEditing={true}
                                        objectType={"Api"}
                                    />
                                    <TranslatedAdvancedInput
                                        features={nameInputFeatures}
                                        isRequired={true}
                                        language={language}
                                        name="name"
                                        title={t("Name")}
                                        placeholder={"Weather Data API..."}
                                    />
                                    <TranslatedAdvancedInput
                                        features={summaryInputFeatures}
                                        isRequired={false}
                                        language={language}
                                        name="summary"
                                        title={t("Summary")}
                                        placeholder={"Provides up-to-date weather forecasts for locations worldwide..."}
                                    />
                                    <TranslatedAdvancedInput
                                        features={detailsInputFeatures}
                                        isRequired={false}
                                        language={language}
                                        name="details"
                                        title={t("Details")}
                                        placeholder={"Detailed weather information including current conditions, hourly forecasts, and daily summaries. Supports JSON and XML formats..."}
                                    />
                                    <LanguageInput
                                        currentLanguage={language}
                                        flexDirection="row-reverse"
                                        handleAdd={handleAddLanguage}
                                        handleDelete={handleDeleteLanguage}
                                        handleCurrent={setLanguage}
                                        languages={languages}
                                    />
                                    <TagSelector name="root.tags" sx={tagSelectorStyle} />
                                    <VersionInput
                                        fullWidth
                                        versions={versions}
                                    />
                                    <ResourceListInput
                                        horizontal
                                        isCreate={true}
                                        parent={resourceListParent}
                                        sxs={resourceListStyle}
                                    />
                                </Box>
                            </Box>
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Divider sx={dividerStyle} />
                            <Box width="100%" padding={2}>
                                <Typography variant="h4" sx={formSectionTitleStyle}>API info</Typography>
                                <Field
                                    fullWidth
                                    name="callLink"
                                    label={"Endpoint URL"}
                                    placeholder={"https://example.com"}
                                    as={TextInput}
                                    InputProps={callLinkInputProps}
                                />
                                <Box width="100%" marginTop={2} color="text.secondary">
                                    <Typography variant="h5" color="text.primary" sx={schemaCollapseStyle.titleContainer}>Schema</Typography>
                                    <MarkdownDisplay
                                        content={"Enter your API's [OpenAPI](https://swagger.io/specification/) or [GraphQL](https://graphql.org/) schema.\nAlternatively, you can enter a link to fetch the schema from."}
                                    />
                                    <Box display="flex" flexDirection="row" gap={1} marginBottom={2}>
                                        <Button
                                            fullWidth
                                            color="secondary"
                                            onClick={toggleHasDocUrlTrue}
                                            variant="outlined"
                                        >{hasDocUrl === true ? "Enter text" : "Use link"}</Button>
                                        <Button
                                            fullWidth
                                            color="secondary"
                                            onClick={showExample}
                                            variant="outlined"
                                            startIcon={<IconCommon name="Help" />}
                                        >Show example</Button>
                                    </Box>
                                </Box>
                                {
                                    hasDocUrl === true && (
                                        <Field
                                            fullWidth
                                            name="documentationLink"
                                            label={"Schema URL (Optional)"}
                                            helperText={"The full URL to the schema"}
                                            as={TextInput}
                                        />
                                    )
                                }
                                {
                                    hasDocUrl === false && (
                                        <CodeInput
                                            codeLanguageField="schemaLanguage"
                                            disabled={false}
                                            limitTo={codeLimitTo}
                                            name="schemaText"
                                        />
                                    )
                                }
                            </Box>
                        </Grid>
                    </Grid>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </MaybeLargeDialog>
    );
}

export function ApiUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ApiUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<ApiVersion, ApiVersionShape>({
        ...endpointsApiVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });

    async function validateValues(values: ApiVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformApiVersionValues, apiVersionValidation);
    }

    const versions = useMemo(function versionsMemo() {
        return (existing?.root as ApiShape)?.versions?.map(v => v.versionLabel) ?? [];
    }, [existing]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Formik
                    enableReinitialize={true}
                    initialValues={existing}
                    onSubmit={noopSubmit}
                    validate={validateValues}
                >
                    {(formik) => <ApiForm
                        disabled={!(isCreate || permissions.canUpdate)}
                        display={display}
                        existing={existing}
                        handleUpdate={setExisting}
                        isCreate={isCreate}
                        isReadLoading={isReadLoading}
                        isOpen={isOpen}
                        versions={versions}
                        {...props}
                        {...formik}
                    />}
                </Formik>
            </ScrollBox>
        </PageContainer>
    );
}
