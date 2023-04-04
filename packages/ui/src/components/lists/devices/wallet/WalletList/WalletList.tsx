/**
 * Displays a list of wallets for the user to manage
 */
import { Box, Button } from '@mui/material';
import { DeleteOneInput, Success, Wallet, WalletUpdateInput } from '@shared/consts';
import { AddIcon } from '@shared/icons';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';
import { walletUpdate } from 'api/generated/endpoints/wallet_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { ListContainer } from 'components/containers/ListContainer/ListContainer';
import { WalletInstallDialog } from 'components/dialogs/WalletInstallDialog/WalletInstallDialog';
import { WalletSelectDialog } from 'components/dialogs/WalletSelectDialog/WalletSelectDialog';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { hasWalletExtension, validateWallet } from 'utils/authentication/walletIntegration';
import { PubSub } from 'utils/pubsub';
import { updateArray } from 'utils/shape/general';
import { WalletListProps } from '../types';
import { WalletListItem } from '../WalletListItem/WalletListItem';

export const WalletList = ({
    handleUpdate,
    numVerifiedEmails,
    list,
}: WalletListProps) => {
    const { t } = useTranslation();

    const [updateMutation, { loading: loadingUpdate }] = useCustomMutation<Wallet, WalletUpdateInput>(walletUpdate);
    const onUpdate = useCallback((index: number, updatedWallet: Wallet) => {
        if (loadingUpdate) return;
        mutationWrapper<Wallet, WalletUpdateInput>({
            mutation: updateMutation,
            input: {
                id: updatedWallet.id,
                name: updatedWallet.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedWallet));
            },
        })
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((wallet: Wallet) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other wallet or one other email)
        if (list.length <= 1 && numVerifiedEmails === 0) {
            PubSub.get().publishSnack({ messageKey: 'MustLeaveVerificationMethod', severity: 'Error' });
            return;
        }
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            messageKey: 'WalletDeleteConfirm',
            messageVariables: { walletName: wallet.name ?? wallet.stakingAddress },
            buttons: [
                {
                    labelKey: 'Yes',
                    onClick: () => {
                        mutationWrapper<Success, DeleteOneInput>({
                            mutation: deleteMutation,
                            input: { id: wallet.id, objectType: 'Wallet' as any },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== wallet.id)])
                            },
                        })
                    }
                },
                { labelKey: 'Cancel', onClick: () => { } },
            ]
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedEmails]);

    // Wallet provider popups
    const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletAddDialog = useCallback(() => {
        setSelectedIndex(null);
        setConnectOpen(true)
    }, []);
    const openWalletVerifyDialog = useCallback((wallet: Wallet) => {
        const index = list.findIndex(w => w.id === wallet.id);
        setSelectedIndex(index);
        setConnectOpen(true)
    }, [list]);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true) }, []);

    /**
     * Add new wallet
     */
    const addWallet = useCallback(async (providerKey: string) => {
        // Check if wallet extension installed
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publishAlertDialog({
                messageKey: 'WalletProviderNotFoundDetails',
                buttons: [
                    { labelKey: 'TryAgain', onClick: () => { addWallet(providerKey); } },
                    { labelKey: 'InstallWallet', onClick: openWalletInstallDialog },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.wallet) {
            // Check if wallet is already in list (i.e. user has already added this wallet)
            const existingWallet = list.find(w => w.stakingAddress === walletCompleteResult.wallet?.stakingAddress);
            if (existingWallet) {
                PubSub.get().publishSnack({ messageKey: 'WalletAlreadyConnected', severity: 'Warning' })
            }
            else {
                PubSub.get().publishSnack({ messageKey: 'WalletVerified', severity: 'Success' });
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
            PubSub.get().publishAlertDialog({
                messageKey: 'WalletProviderNotFoundDetails',
                buttons: [
                    { labelKey: 'TryAgain', onClick: () => { verifyWallet(providerKey); } },
                    { labelKey: 'InstallWallet', onClick: openWalletInstallDialog },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult) {
            PubSub.get().publishSnack({ messageKey: 'WalletVerified', severity: 'Success' })
            // Update list
            handleUpdate(updateArray(list, selectedIndex, {
                ...list[selectedIndex],
                verified: true,
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

    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false) }, []);

    return (
        <>
            {/* Select dialog for adding new wallet */}
            <WalletSelectDialog
                handleOpenInstall={openWalletInstallDialog}
                open={connectOpen}
                onClose={closeWalletConnectDialog}
                zIndex={200}
            />
            {/* Install dialog for downloading wallet extension */}
            <WalletInstallDialog
                open={installOpen}
                onClose={closeWalletInstallDialog}
                zIndex={connectOpen ? 201 : 200}
            />
            <ListContainer
                emptyText={t(`NoWallets`, { ns: 'error' })}
                isEmpty={list.length === 0}
                sx={{ maxWidth: '500px' }}
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
            <Box id='add-wallet-button' sx={{
                alignItems: 'center',
                display: 'flex',
                justifyContent: 'center',
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                <Button
                    fullWidth
                    onClick={openWalletAddDialog}
                    startIcon={<AddIcon />}
                    sx={{
                        maxWidth: '400px',
                        width: 'auto',
                    }}
                >Add Wallet</Button>
            </Box>
        </>
    )
}