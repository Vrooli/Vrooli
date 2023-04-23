import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon } from "@local/icons";
import { Box, Button } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { deleteOneOrManyDeleteOne } from "../../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { walletUpdate } from "../../../../../api/generated/endpoints/wallet_update";
import { useCustomMutation } from "../../../../../api/hooks";
import { mutationWrapper } from "../../../../../api/utils";
import { hasWalletExtension, validateWallet } from "../../../../../utils/authentication/walletIntegration";
import { PubSub } from "../../../../../utils/pubsub";
import { updateArray } from "../../../../../utils/shape/general";
import { ListContainer } from "../../../../containers/ListContainer/ListContainer";
import { WalletInstallDialog } from "../../../../dialogs/WalletInstallDialog/WalletInstallDialog";
import { WalletSelectDialog } from "../../../../dialogs/WalletSelectDialog/WalletSelectDialog";
import { WalletListItem } from "../WalletListItem/WalletListItem";
export const WalletList = ({ handleUpdate, numVerifiedEmails, list, }) => {
    const { t } = useTranslation();
    const [updateMutation, { loading: loadingUpdate }] = useCustomMutation(walletUpdate);
    const onUpdate = useCallback((index, updatedWallet) => {
        if (loadingUpdate)
            return;
        mutationWrapper({
            mutation: updateMutation,
            input: {
                id: updatedWallet.id,
                name: updatedWallet.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedWallet));
            },
        });
    }, [handleUpdate, list, loadingUpdate, updateMutation]);
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((wallet) => {
        if (loadingDelete)
            return;
        if (list.length <= 1 && numVerifiedEmails === 0) {
            PubSub.get().publishSnack({ messageKey: "MustLeaveVerificationMethod", severity: "Error" });
            return;
        }
        PubSub.get().publishAlertDialog({
            messageKey: "WalletDeleteConfirm",
            messageVariables: { walletName: wallet.name ?? wallet.stakingAddress },
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: wallet.id, objectType: "Wallet" },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== wallet.id)]);
                            },
                        });
                    },
                },
                { labelKey: "Cancel", onClick: () => { } },
            ],
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedEmails]);
    const [selectedIndex, setSelectedIndex] = useState(null);
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletAddDialog = useCallback(() => {
        setSelectedIndex(null);
        setConnectOpen(true);
    }, []);
    const openWalletVerifyDialog = useCallback((wallet) => {
        const index = list.findIndex(w => w.id === wallet.id);
        setSelectedIndex(index);
        setConnectOpen(true);
    }, [list]);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);
    const addWallet = useCallback(async (providerKey) => {
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publishAlertDialog({
                messageKey: "WalletProviderNotFoundDetails",
                buttons: [
                    { labelKey: "TryAgain", onClick: () => { addWallet(providerKey); } },
                    { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
                ],
            });
            return;
        }
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.wallet) {
            const existingWallet = list.find(w => w.stakingAddress === walletCompleteResult.wallet?.stakingAddress);
            if (existingWallet) {
                PubSub.get().publishSnack({ messageKey: "WalletAlreadyConnected", severity: "Warning" });
            }
            else {
                PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
                handleUpdate([...list, walletCompleteResult.wallet]);
            }
        }
    }, [handleUpdate, list, openWalletInstallDialog]);
    const verifyWallet = useCallback(async (providerKey) => {
        if (selectedIndex === null)
            return;
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publishAlertDialog({
                messageKey: "WalletProviderNotFoundDetails",
                buttons: [
                    { labelKey: "TryAgain", onClick: () => { verifyWallet(providerKey); } },
                    { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
                ],
            });
            return;
        }
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult) {
            PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
            handleUpdate(updateArray(list, selectedIndex, {
                ...list[selectedIndex],
                verified: true,
            }));
        }
    }, [handleUpdate, list, openWalletInstallDialog, selectedIndex]);
    const closeWalletConnectDialog = useCallback((providerKey) => {
        setConnectOpen(false);
        const index = selectedIndex;
        setSelectedIndex(null);
        if (providerKey) {
            if (!index || index <= 0) {
                addWallet(providerKey);
            }
            else {
                verifyWallet(providerKey);
            }
        }
    }, [addWallet, selectedIndex, verifyWallet]);
    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);
    return (_jsxs(_Fragment, { children: [_jsx(WalletSelectDialog, { handleOpenInstall: openWalletInstallDialog, open: connectOpen, onClose: closeWalletConnectDialog, zIndex: 200 }), _jsx(WalletInstallDialog, { open: installOpen, onClose: closeWalletInstallDialog, zIndex: connectOpen ? 201 : 200 }), _jsx(ListContainer, { emptyText: t("NoWallets", { ns: "error" }), isEmpty: list.length === 0, sx: { maxWidth: "500px" }, children: list.map((w, index) => (_jsx(WalletListItem, { data: w, index: index, handleUpdate: onUpdate, handleDelete: onDelete, handleVerify: openWalletVerifyDialog }, `wallet-${index}`))) }), _jsx(Box, { id: 'add-wallet-button', sx: {
                    alignItems: "center",
                    display: "flex",
                    justifyContent: "center",
                    paddingTop: 2,
                    paddingBottom: 6,
                }, children: _jsx(Button, { fullWidth: true, onClick: openWalletAddDialog, startIcon: _jsx(AddIcon, {}), sx: {
                        maxWidth: "400px",
                        width: "auto",
                    }, children: "Add Wallet" }) })] }));
};
//# sourceMappingURL=WalletList.js.map