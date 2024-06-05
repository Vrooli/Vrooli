import { CodeType, CodeVersion, CodeVersionCreateInput, CodeVersionUpdateInput, DUMMY_ID, LINKS, Session, codeVersionTranslationValidation, codeVersionValidation, endpointGetCodeVersion, endpointPostCodeVersion, endpointPutCodeVersion, noopSubmit, orDefault } from "@local/shared";
import { Button, Divider, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { HelpIcon, SearchIcon } from "icons";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { CodeShape } from "utils/shape/models/code";
import { CodeVersionShape, shapeCodeVersion } from "utils/shape/models/codeVersion";
import { validateFormValues } from "utils/validateFormValues";
import { SmartContractFormProps, SmartContractUpsertProps } from "../types";

const smartContractInitialValues = (
    session: Session | undefined,
    existing?: Partial<CodeVersion> | undefined,
): CodeVersionShape => ({
    __typename: "CodeVersion" as const,
    id: DUMMY_ID,
    codeLanguage: CodeLanguage.Javascript,
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
});

const transformCodeVersionValues = (values: CodeVersionShape, existing: CodeVersionShape, isCreate: boolean) =>
    isCreate ? shapeCodeVersion.create(values) : shapeCodeVersion.update(existing, values);

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

const SmartContractForm = ({
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
}: SmartContractFormProps) => {
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

    const { handleCancel, handleCompleted } = useUpsertActions<CodeVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "CodeVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<CodeVersion, CodeVersionCreateInput, CodeVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostCodeVersion,
        endpointUpdate: endpointPutCodeVersion,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "CodeVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

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
    const showExample = () => {
        // Determine example to show based on current language
        const exampleCode = codeLanguageField.value === CodeLanguage.Haskell ? examplePlutusContract : exampleSolidityContract;
        // Set value to hard-coded example
        contentHelpers.setValue(exampleCode);
    };

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
            <Button
                href={`${LINKS.Search}?type=${SearchPageTabOption.Code}`}
                sx={{
                    color: palette.background.textSecondary,
                    display: "flex",
                    marginTop: 2,
                    textAlign: "center",
                    textTransform: "none",
                }}
                variant="text"
                endIcon={<SearchIcon />}
            >
                Search existing codes
            </Button>
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Code"}
                            sx={{ marginBottom: 2 }}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={{ __typename: "CodeVersion", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
                        />
                        <FormSection sx={{ overflowX: "hidden", marginBottom: 2 }}>
                            <TranslatedTextInput
                                autoFocus
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
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                                sx={{ flexDirection: "row-reverse" }}
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
                                    sx={{ marginLeft: 2 }}
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
                            limitTo={[CodeLanguage.Solidity, CodeLanguage.Haskell]}
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
            />
        </MaybeLargeDialog>
    );
};

export const SmartContractUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: SmartContractUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<CodeVersion, CodeVersionShape>({
        ...endpointGetCodeVersion,
        isCreate,
        objectType: "CodeVersion",
        overrideObject,
        transform: (existing) => smartContractInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformCodeVersionValues, codeVersionValidation)}
        >
            {(formik) => <SmartContractForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={(existing?.root as CodeShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
