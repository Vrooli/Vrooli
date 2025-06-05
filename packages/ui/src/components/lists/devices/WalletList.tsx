import { Box, Button, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { endpointsActions, endpointsWallet, updateArray, type DeleteOneInput, type Success, type Wallet, type WalletUpdateInput } from "@vrooli/shared";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { WalletInstallDialog, WalletSelectDialog } from "../../../components/dialogs/auth.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { multiLineEllipsis } from "../../../styles.js";
import { hasWalletExtension, validateWallet } from "../../../utils/authentication/walletIntegration.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { type WalletListItemProps, type WalletListProps } from "./types.js";

export function WalletListItem({
    handleDelete,
    handleUpdate,
    handleVerify,
    index,
    data,
}: WalletListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const isNamed = useMemo(() => data.name && data.name.length > 0, [data.name]);

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    const onVerify = useCallback(() => {
        handleVerify(data);
    }, [data, handleVerify]);

    /**
     * Shortens staking address to first 2 letters, an ellipsis, and last 6 letters
     */
    const shortenedAddress = useMemo(() => {
        if (!data.stakingAddress) return "";
        return `${data.stakingAddress.substring(0, 2)}...${data.stakingAddress.substring(data.stakingAddress.length - 6)}`;
    }, [data.stakingAddress]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: "flex",
                padding: 1,
                borderBottom: `1px solid ${palette.divider}`,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} mr="auto">
                <Stack direction="row" spacing={1}>
                    {/* Name (or publich address if not name) */}
                    <ListItemText
                        primary={isNamed ? data.name : shortenedAddress}
                        sx={{ ...multiLineEllipsis(1) }}
                    />
                    {/* Bio/Description */}
                    {isNamed && <ListItemText
                        primary={shortenedAddress}
                        sx={{ ...multiLineEllipsis(1), color: palette.text.secondary }}
                    />}
                </Stack>
                {/* Verified indicator */}
                <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? palette.success.main : palette.error.main}`,
                    color: data.verified ? palette.success.main : palette.error.main,
                    height: "fit-content",
                    fontWeight: "bold",
                    marginTop: "auto",
                    marginBottom: "auto",
                    textAlign: "center",
                    padding: 0.25,
                    width: "fit-content",
                }}>
                    {t(data.verified ? "Verified" : "VerifiedNot")}
                </Box>
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                {!data.verified && <Tooltip title={t("VerifyWallet")}>
                    <IconButton
                        onClick={onVerify}
                    >
                        <IconCommon
                            decorative
                            fill="error.main"
                            name="Complete"
                        />
                    </IconButton>
                </Tooltip>}
                <Tooltip title={t("WalletDelete")}>
                    <IconButton
                        onClick={onDelete}
                    >
                        <IconCommon
                            decorative
                            fill="secondary.main"
                            name="Delete"
                        />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}

/**
 * Displays a list of wallets for the user to manage
 */
export function WalletList({
    handleUpdate,
    numOtherVerified,
    list,
}: WalletListProps) {
    const { t } = useTranslation();

    const [updateMutation, { loading: loadingUpdate }] = useLazyFetch<WalletUpdateInput, Wallet>(endpointsWallet.updateOne);
    const onUpdate = useCallback((index: number, updatedWallet: Wallet) => {
        if (loadingUpdate) return;
        fetchLazyWrapper<WalletUpdateInput, Wallet>({
            fetch: updateMutation,
            inputs: {
                id: updatedWallet.id,
                name: updatedWallet.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedWallet));
            },
        });
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const onDelete = useCallback((wallet: Wallet) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        if (list.length <= 1 && numOtherVerified === 0) {
            PubSub.get().publish("snack", { messageKey: "MustLeaveVerificationMethod", severity: "Error" });
            return;
        }
        // Confirmation dialog
        PubSub.get().publish("alertDialog", {
            messageKey: "WalletDeleteConfirm",
            messageVariables: { walletName: wallet.name ?? wallet.stakingAddress },
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        fetchLazyWrapper<DeleteOneInput, Success>({
                            fetch: deleteMutation,
                            inputs: { id: wallet.id, objectType: "Wallet" as any },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== wallet.id)]);
                            },
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numOtherVerified]);

    // Wallet provider popups
    const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletAddDialog = useCallback(() => {
        setSelectedIndex(null);
        setConnectOpen(true);
    }, []);
    const openWalletVerifyDialog = useCallback((wallet: Wallet) => {
        const index = list.findIndex(w => w.id === wallet.id);
        setSelectedIndex(index);
        setConnectOpen(true);
    }, [list]);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);

    /**
     * Add new wallet
     */
    const addWallet = useCallback(async (providerKey: string) => {
        // Check if wallet extension installed
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publish("alertDialog", {
                messageKey: "WalletProviderNotFoundDetails",
                buttons: [
                    { labelKey: "TryAgain", onClick: () => { addWallet(providerKey); } },
                    { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
                ],
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.wallet) {
            // Check if wallet is already in list (i.e. user has already added this wallet)
            const existingWallet = list.find(w => w.stakingAddress === walletCompleteResult.wallet?.stakingAddress);
            if (existingWallet) {
                PubSub.get().publish("snack", { messageKey: "WalletAlreadyConnected", severity: "Warning" });
            }
            else {
                PubSub.get().publish("snack", { messageKey: "WalletVerified", severity: "Success" });
                // Update list
                handleUpdate([...list, walletCompleteResult.wallet]);
            }
        }
    }, [handleUpdate, list, openWalletInstallDialog]);

    /**
     * Verify existing wallet
     */
    const verifyWallet = useCallback(async (providerKey: string) => {
        if (selectedIndex === null) return;
        // Check if wallet extension installed
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publish("alertDialog", {
                messageKey: "WalletProviderNotFoundDetails",
                buttons: [
                    { labelKey: "TryAgain", onClick: () => { verifyWallet(providerKey); } },
                    { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
                ],
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult) {
            PubSub.get().publish("snack", { messageKey: "WalletVerified", severity: "Success" });
            // Update list
            handleUpdate(updateArray(list, selectedIndex, {
                ...list[selectedIndex],
                verifiedAt: new Date(),
            }));
        }
    }, [handleUpdate, list, openWalletInstallDialog, selectedIndex]);

    const closeWalletConnectDialog = useCallback((providerKey: string | null) => {
        setConnectOpen(false);
        const index = selectedIndex;
        setSelectedIndex(null);
        if (providerKey) {
            if (!index || index <= 0) {
                addWallet(providerKey);
            } else {
                verifyWallet(providerKey);
            }
        }
    }, [addWallet, selectedIndex, verifyWallet]);

    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);

    return (
        <>
            {/* Select dialog for adding new wallet */}
            <WalletSelectDialog
                handleOpenInstall={openWalletInstallDialog}
                open={connectOpen}
                onClose={closeWalletConnectDialog}
            />
            {/* Install dialog for downloading wallet extension */}
            <WalletInstallDialog
                open={installOpen}
                onClose={closeWalletInstallDialog}
            />
            <ListContainer
                emptyText={t("NoWallets", { ns: "error" })}
                isEmpty={list.length === 0}
            >
                {/* Wallet list */}
                {list.map((w: Wallet, index) => (
                    <WalletListItem
                        key={`wallet-${index}`}
                        data={w}
                        index={index}
                        handleUpdate={onUpdate}
                        handleDelete={onDelete}
                        handleVerify={openWalletVerifyDialog}
                    />
                ))}
            </ListContainer>
            {/* Add new button */}
            <Box
                id='add-wallet-button'
                alignItems="center"
                display="flex"
                justifyContent="center"
                paddingTop={4}
            >
                <Button
                    fullWidth
                    onClick={openWalletAddDialog}
                    startIcon={<IconCommon
                        decorative
                        name="Add"
                    />}
                    variant="outlined"
                >{t("AddWallet")}</Button>
            </Box>
        </>
    );
}
