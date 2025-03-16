import { CodeLanguage, CodeShape, CodeType, CodeVersion, CodeVersionCreateInput, CodeVersionShape, CodeVersionUpdateInput, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, Session, codeVersionTranslationValidation, codeVersionValidation, endpointsCodeVersion, noopSubmit, orDefault, shapeCodeVersion } from "@local/shared";
import { Button, Divider, useTheme } from "@mui/material";
import { useSubmitHelper } from "api/fetchWrapper.js";
import { AutoFillButton } from "components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton.js";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse.js";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { CodeInput } from "components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput.js";
import { TagSelector } from "components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput.js";
import { VersionInput } from "components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "components/lists/ResourceList/ResourceList.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "hooks/forms.js";
import { UseAutoFillProps, createUpdatedTranslations, getAutoFillTranslationData, useAutoFill } from "hooks/tasks.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useTranslatedFields } from "hooks/useTranslatedFields.js";
import { useUpsertFetch } from "hooks/useUpsertFetch.js";
import { HelpIcon } from "icons/common.js";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools.js";
import { validateFormValues } from "utils/validateFormValues.js";
import { SessionContext } from "../../../contexts.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { SmartContractFormProps, SmartContractUpsertProps } from "./types.js";

export function smartContractInitialValues(
    session: Session | undefined,
    existing?: Partial<CodeVersion> | undefined,
): CodeVersionShape {
    return {
        __typename: "CodeVersion" as const,
        id: DUMMY_ID,
        calledByRoutineVersionsCount: 0,
        codeLanguage: CodeLanguage.Solidity,
        codeType: CodeType.SmartContract,
        content: "",
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        resourceList: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "CodeVersion" as const,
                id: DUMMY_ID,
            },
        },
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Code" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "CodeVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            jsonVariable: "",
            name: "",
        }]),
    };
}

function transformCodeVersionValues(values: CodeVersionShape, existing: CodeVersionShape, isCreate: boolean) {
    return isCreate ? shapeCodeVersion.create(values) : shapeCodeVersion.update(existing, values);
}

const examplePlutusContract = `
data AuctionParams = AuctionParams
  { apSeller  :: PubKeyHash,
    -- ^ Seller's wallet address. The highest bid (if exists) will be sent to the seller.
    -- If there is no bid, the asset auctioned will be sent to the seller.
    apAsset   :: Value,
    -- ^ The asset being auctioned. It can be a single token, multiple tokens of the same
    -- kind, or tokens of different kinds, and the token(s) can be fungible or non-fungible.
    -- These can all be encoded as a \`Value\`.
    apMinBid  :: Lovelace,
    -- ^ The minimum bid in Lovelace.
    apEndTime :: POSIXTime
    -- ^ The deadline for placing a bid. This is the earliest time the auction can be closed.
  }

PlutusTx.makeLift ''AuctionParams

data Bid = Bid
  { bBidder :: PubKeyHash,
    -- ^ Bidder's wallet address.
    bAmount :: Lovelace
    -- ^ Bid amount in Lovelace.
  }

PlutusTx.deriveShow ''Bid
PlutusTx.unstableMakeIsData ''Bid

instance PlutusTx.Eq Bid where
  {-# INLINEABLE (==) #-}
  bid == bid' =
    bBidder bid PlutusTx.== bBidder bid'
      PlutusTx.&& bAmount bid PlutusTx.== bAmount bid'

-- | Datum represents the state of a smart contract. In this case
-- it contains the highest bid so far (if exists).
newtype AuctionDatum = AuctionDatum { adHighestBid :: Maybe Bid }

PlutusTx.unstableMakeIsData ''AuctionDatum

-- | Redeemer is the input that changes the state of a smart contract.
-- In this case it is either a new bid, or a request to close the auction
-- and pay out the seller and the highest bidder.
data AuctionRedeemer = NewBid Bid | Payout
`.trim();

const exampleSolidityContract = `
// Specifies the version of Solidity, using semantic versioning.
// Learn more: https://solidity.readthedocs.io/en/v0.5.10/layout-of-source-files.html#pragma
pragma solidity ^0.7.0;

// Defines a contract named \`HelloWorld\`.
// A contract is a collection of functions and data (its state). Once deployed, a contract resides at a specific address on the Ethereum blockchain. Learn more: https://solidity.readthedocs.io/en/v0.5.10/structure-of-a-contract.html
contract HelloWorld {

   // Declares a state variable \`message\` of type \`string\`.
   // State variables are variables whose values are permanently stored in contract storage. The keyword \`public\` makes variables accessible from outside a contract and creates a function that other contracts or clients can call to access the value.
   string public message;

   // Similar to many class-based object-oriented languages, a constructor is a special function that is only executed upon contract creation.
   // Constructors are used to initialize the contract's data. Learn more:https://solidity.readthedocs.io/en/v0.5.10/contracts.html#constructors
   constructor(string memory initMessage) {

      // Accepts a string argument \`initMessage\` and sets the value into the contract's \`message\` storage variable).
      message = initMessage;
   }

   // A public function that accepts a string argument and updates the \`message\` storage variable.
   function update(string memory newMessage) public {
      message = newMessage;
   }
}
`.trim();

const codeLimitTo = [CodeLanguage.Solidity, CodeLanguage.Haskell] as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;

function SmartContractForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    versions,
    ...props
}: SmartContractFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();

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
        validationSchema: codeVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "CodeVersion", id: values.id } as const;
    }, [values]);

    const { handleCancel, handleCompleted } = useUpsertActions<CodeVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "CodeVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<CodeVersion, CodeVersionCreateInput, CodeVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsCodeVersion.createOne,
        endpointUpdate: endpointsCodeVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "CodeVersion" });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            codeLanguage: values.codeLanguage,
            content: values.content,
            isPrivate: values.isPrivate,
            version: values.versionLabel,
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["name", "description"]);
        delete rest.id;
        const codeLanguage = typeof rest.codeLanguage === "string" ? rest.codeLanguage : values.codeLanguage;
        const content = typeof rest.content === "string" ? rest.content : values.content;
        const isPrivate = typeof rest.isPrivate === "boolean" ? rest.isPrivate : values.isPrivate;
        const versionLabel = typeof rest.version === "string" ? rest.version : values.versionLabel;
        const updatedValues = {
            ...values,
            codeLanguage,
            content,
            isPrivate,
            translations: updatedTranslations,
            versionLabel,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.SmartContractAdd : LlmTask.SmartContractUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<CodeVersionCreateInput | CodeVersionUpdateInput, CodeVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformCodeVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const [codeLanguageField, , codeLanguageHelpers] = useField<CodeLanguage>("codeLanguage");
    const [contentField, , contentHelpers] = useField<string>("content");
    const showExample = useCallback(function showExampleCallback() {
        // Determine example to show based on current language
        const exampleCode = codeLanguageField.value === CodeLanguage.Haskell ? examplePlutusContract : exampleSolidityContract;
        // Set value to hard-coded example
        contentHelpers.setValue(exampleCode);
    }, [codeLanguageField.value, contentHelpers]);

    return (
        <MaybeLargeDialog
            display={display}
            id="smart-contract-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateSmartContract" : "UpdateSmartContract")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.SmartContract}"`}
                text="Search existing contracts"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Code"}
                            sx={relationshipListStyle}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={resourceListParent}
                            sxs={resourceListStyle}
                        />
                        <FormSection sx={formSectionStyle}>
                            <TranslatedTextInput
                                fullWidth
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="description"
                                maxChars={2048}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("DescriptionPlaceholder")}
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
                        <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={{ marginBottom: 2 }}
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse
                        title="Contract"
                        titleVariant="h4"
                        isOpen={true}
                        toTheRight={
                            <>
                                <Button
                                    variant="outlined"
                                    onClick={showExample}
                                    startIcon={<HelpIcon />}
                                    sx={exampleButtonStyle}
                                >
                                    Show example
                                </Button>
                                {/* <IconButton
                                onClick={toggleFullscreen}
                                sx={{ marginLeft: 2 }}
                            >
                                {isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
                            </IconButton> */}
                            </>
                        }
                        sxs={{ titleContainer: { marginBottom: 1 } }}
                    >
                        <CodeInput
                            disabled={false}
                            limitTo={codeLimitTo}
                            name="content"
                        />
                    </ContentCollapse>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
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

export function SmartContractUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: SmartContractUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<CodeVersion, CodeVersionShape>({
        ...endpointsCodeVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "CodeVersion",
        overrideObject,
        transform: (existing) => smartContractInitialValues(session, existing),
    });

    async function validateValues(values: CodeVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformCodeVersionValues, codeVersionValidation);
    }

    const versions = useMemo(function versionsMemo() {
        return (existing?.root as CodeShape)?.versions?.map(v => v.versionLabel) ?? [];
    }, [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <SmartContractForm
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
    );
}
