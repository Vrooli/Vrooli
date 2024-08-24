import { ApiShape, ApiVersion, ApiVersionCreateInput, ApiVersionShape, apiVersionTranslationValidation, ApiVersionUpdateInput, apiVersionValidation, AutoFillResult, CodeLanguage, DUMMY_ID, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion, LINKS, LlmTask, noopSubmit, orDefault, SearchPageTabOption, Session, shapeApiVersion } from "@local/shared";
import { Button, Divider, InputAdornment, Stack } from "@mui/material";
import { useSubmitHelper } from "api";
import { AutoFillButton } from "components/buttons/AutoFillButton/AutoFillButton";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TextInput, TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill } from "hooks/useAutoFill";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { HelpIcon, LinkIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateFormValues } from "utils/validateFormValues";
import { SearchExistingButton } from "../../../../components/buttons/SearchExistingButton/SearchExistingButton";
import { ApiFormProps, ApiUpsertProps } from "../types";

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
            <LinkIcon />
        </InputAdornment>
    ),
} as const;
const codeLimitTo = [CodeLanguage.Json, CodeLanguage.Graphql, CodeLanguage.Yaml] as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const basicInfoCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const versionInputStyle = { marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const apiInfoCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;
const schemaCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;

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
        endpointCreate: endpointPostApiVersion,
        endpointUpdate: endpointPutApiVersion,
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

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback({ data }: AutoFillResult) {
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
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={basicInfoCollapseStyle}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Api"}
                            sx={relationshipListStyle}
                        />
                        <FormSection sx={formSectionStyle}>
                            <TranslatedTextInput
                                fullWidth
                                isRequired={true}
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                isRequired={false}
                                language={language}
                                name="summary"
                                maxChars={1024}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("Summary")}
                            />
                            <TranslatedRichInput
                                isRequired={false}
                                language={language}
                                name="details"
                                maxChars={8192}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("Details")}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                flexDirection="row-reverse"
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                            />
                        </FormSection>
                        <TagSelector name="root.tags" sx={tagSelectorStyle} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={versionInputStyle}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={resourceListParent}
                            sxs={resourceListStyle}
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse title="Api info" titleVariant="h4" isOpen={true} sxs={apiInfoCollapseStyle}>
                        <FormSection>
                            <Field
                                fullWidth
                                name="callLink"
                                label={"Endpoint URL"}
                                placeholder={"https://example.com"}
                                as={TextInput}
                                InputProps={callLinkInputProps}
                            />
                            {/* Selector for documentation URL or text */}
                            <ContentCollapse
                                title="Schema"
                                titleVariant="h4"
                                helpText={"Enter your API's [OpenAPI](https://swagger.io/specification/) or [GraphQL](https://graphql.org/) schema.\n\nAlternatively, you can enter a link to fetch the schema from."}
                                isOpen={true}
                                sxs={schemaCollapseStyle}
                            >
                                <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1} >
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
                                        startIcon={<HelpIcon />}
                                    >Show example</Button>
                                </Stack >
                            </ContentCollapse>
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
                        </FormSection>
                    </ContentCollapse>
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
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ApiUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
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
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ApiForm
                disabled={!(isCreate || permissions.canUpdate)}
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
    );
}
