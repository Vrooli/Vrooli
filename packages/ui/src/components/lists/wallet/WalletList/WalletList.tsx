/**
 * Displays a list of wallets for the user to manage
 */
import { WalletListProps } from '../types';
import { useCallback, useState } from 'react';
import { Wallet } from 'types';
import { Box, Button } from '@mui/material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { PubSub, updateArray } from 'utils';
import { deleteOneMutation, walletUpdateMutation } from 'graphql/mutation';
import { hasWalletExtension, validateWallet } from 'utils/authentication/walletIntegration';
import { WalletListItem } from '../WalletListItem/WalletListItem';
import { DeleteOneType } from '@shared/consts';
import { deleteOneVariables, deleteOne_deleteOne } from 'graphql/generated/deleteOne';
import { walletUpdateVariables, walletUpdate_walletUpdate } from 'graphql/generated/walletUpdate';
import { SnackSeverity, WalletInstallDialog, WalletSelectDialog } from 'components';
import { AddIcon } from '@shared/icons';

export const WalletList = ({
    handleUpdate,
    numVerifiedEmails,
    list,
}: WalletListProps) => {

    const [updateMutation, { loading: loadingUpdate }] = useMutation(walletUpdateMutation);
    const onUpdate = useCallback((index: number, updatedWallet: Wallet) => {
        if (loadingUpdate) return;
        mutationWrapper<walletUpdate_walletUpdate, walletUpdateVariables>({
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

    const [deleteMutation, { loading: loadingDelete }] = useMutation(deleteOneMutation);
    const onDelete = useCallback((wallet: Wallet) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other wallet or one other email)
        if (list.length <= 1 && numVerifiedEmails === 0) {
            PubSub.get().publishSnack({ message: 'Cannot delete your only authentication method!', severity: SnackSeverity.Error });
            return;
        }
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            message: `Are you sure you want to delete wallet ${wallet.name ?? wallet.stakingAddress}?`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper<deleteOne_deleteOne, deleteOneVariables>({
                            mutation: deleteMutation,
                            input: { id: wallet.id, objectType: DeleteOneType.Wallet },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== wallet.id)])
                            },
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
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
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave) that supports extensions, and that your wallet extension is enabled.',
                buttons: [
                    { text: 'Try Again', onClick: () => { addWallet(providerKey); } },
                    { text: 'Install Wallet', onClick: openWalletInstallDialog },
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
                PubSub.get().publishSnack({ message: 'Wallet already connected.', severity: SnackSeverity.Warning })
            }
            else {
                PubSub.get().publishSnack({ message: 'Wallet verified.', severity: SnackSeverity.Success });
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
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave) that supports extensions, and that your wallet extension is enabled.',
                buttons: [
                    { text: 'Try Again', onClick: () => { verifyWallet(providerKey); } },
                    { text: 'Install Wallet', onClick: openWalletInstallDialog },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult) {
            PubSub.get().publishSnack({ message: 'Wallet verified.', severity: SnackSeverity.Success })
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
            <Box sx={{
                overflow: 'overlay',
                border: `1px solid #e0e0e0`,
                borderRadius: '8px',
                maxWidth: '1000px',
                marginLeft: 1,
                marginRight: 1,
            }}>
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
            </Box>
            {/* Add new button */}
            <Box sx={{
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