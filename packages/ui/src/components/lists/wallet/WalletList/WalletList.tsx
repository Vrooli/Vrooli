/**
 * Displays a list of wallets for the user to manage
 */
import { WalletListProps } from '../types';
import { useCallback, useState } from 'react';
import { Wallet } from 'types';
import { Box, Button, Dialog, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { Pubs, updateArray } from 'utils';
import { deleteOneMutation, walletUpdateMutation } from 'graphql/mutation';
import { hasWalletExtension, validateWallet, WalletProvider, walletProviderInfo } from 'utils/authentication/walletIntegration';
import { WalletListItem } from '../WalletListItem/WalletListItem';
import { DeleteOneType } from '@local/shared';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { walletUpdate, walletUpdateVariables } from 'graphql/generated/walletUpdate';

export const WalletList = ({
    handleUpdate,
    numVerifiedEmails,
    list,
}: WalletListProps) => {
    const { palette } = useTheme();

    const [updateMutation, { loading: loadingUpdate }] = useMutation<walletUpdate, walletUpdateVariables>(walletUpdateMutation);
    const onUpdate = useCallback((index: number, updatedWallet: Wallet) => {
        if (loadingUpdate) return;
        mutationWrapper({
            mutation: updateMutation,
            input: {
                id: updatedWallet.id,
                name: updatedWallet.name,
            },
            onSuccess: (response) => {
                handleUpdate(updateArray(list, index, updatedWallet));
            },
        })
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation);
    const onDelete = useCallback((wallet: Wallet) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other wallet or one other email)
        if (list.length <= 1 && numVerifiedEmails === 0) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot delete your only authentication method!', severity: 'error' });
            return;
        }
        // Confirmation dialog
        PubSub.publish(Pubs.AlertDialog, {
            message: `Are you sure you want to delete wallet ${wallet.name ?? wallet.stakingAddress}?`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: wallet.id, objectType: DeleteOneType.Wallet },
                            onSuccess: (response) => {
                                handleUpdate([...list.filter(w => w.id !== wallet.id)])
                            },
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedEmails]);

    // Opens link to install wallet extension
    const downloadExtension = useCallback((provider: WalletProvider) => {
        const extensionUrl = walletProviderInfo[provider].extensionUrl;
        window.open(extensionUrl, '_blank', 'noopener,noreferrer');
    }, [])

    // Wallet provider select popup
    const [providerOpen, setproviderOpen] = useState(false);
    const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
    const [walletDialogFor, setWalletDialogFor] = useState<'add' | 'verify' | 'download'>('add');
    const openProviderAddDialog = useCallback(() => {
        setWalletDialogFor('add');
        setproviderOpen(true);
    }, []);
    const openProviderVerifyDialog = useCallback((wallet: Wallet) => {
        const index = list.findIndex(w => w.id === wallet.id);
        setWalletDialogFor('verify');
        setSelectedIndex(index);
        setproviderOpen(true);
    }, [list]);
    const openProviderDownloadDialog = useCallback(() => {
        setWalletDialogFor('download');
        setproviderOpen(true);
    }, []);

    /**
     * Add new wallet
     */
    const addWallet = useCallback(async (provider: WalletProvider) => {
        // Check if wallet extension installed
        if (!hasWalletExtension(provider)) {
            PubSub.publish(Pubs.AlertDialog, {
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: () => { addWallet(provider); } },
                    { text: 'Install Wallet', onClick: openProviderDownloadDialog },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(provider);
        if (walletCompleteResult?.wallet) {
            // Check if wallet is already in list (i.e. user has already added this wallet)
            const existingWallet = list.find(w => w.stakingAddress === walletCompleteResult.wallet?.stakingAddress);
            if (existingWallet) {
                PubSub.publish(Pubs.Snack, { message: 'Wallet already connected.', severity: 'warning' })
            }
            else {
                PubSub.publish(Pubs.Snack, { message: 'Wallet verified.', severity: 'success' });
                // Update list
                handleUpdate([...list, walletCompleteResult.wallet]);
            }
        }
    }, [handleUpdate, list, openProviderDownloadDialog]);

    /**
     * Verify existing wallet
     */
    const verifyWallet = useCallback(async (provider: WalletProvider) => {
        if (selectedIndex === null) return;
        // Check if wallet extension installed
        if (!hasWalletExtension(provider)) {
            PubSub.publish(Pubs.AlertDialog, {
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: () => { verifyWallet(provider); } },
                    { text: 'Install Wallet', onClick: openProviderDownloadDialog },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(provider);
        if (walletCompleteResult) {
            PubSub.publish(Pubs.Snack, { message: 'Wallet verified.', severity: 'success' })
            // Update list
            handleUpdate(updateArray(list, selectedIndex, {
                ...list[selectedIndex],
                verified: true,
            }));
        }
    }, [handleUpdate, list, openProviderDownloadDialog, selectedIndex]);

    const handleProviderClose = useCallback(() => {
        setproviderOpen(false);
    }, [])
    const handleProviderSelect = useCallback((selected: WalletProvider) => {
        if (walletDialogFor === 'add') {
            addWallet(selected);
        } else if (walletDialogFor === 'verify') {
            verifyWallet(selected);
        } else if (walletDialogFor === 'download') {
            downloadExtension(selected);
        }
        handleProviderClose();
    }, [addWallet, downloadExtension, handleProviderClose, verifyWallet, walletDialogFor])

    return (
        <>
            {/* Popup for selecting wallet provider (to add new wallet) */}
            <Dialog
                open={providerOpen}
                disableScrollLock={true}
                onClose={handleProviderClose}
            >
                <Box
                    sx={{
                        width: '100',
                        borderRadius: '4px 4px 0 0',
                        padding: 1,
                        paddingLeft: 2,
                        paddingRight: 2,
                        background: palette.primary.dark,
                        color: 'white',
                    }}
                >
                    <Typography variant="h6" textAlign="center">Select Wallet</Typography>
                </Box>
                {Object.values(walletProviderInfo).map((o, index) => (
                    <ListItem button onClick={() => handleProviderSelect(o.enum)} key={index}>
                        <ListItemText primary={o.label} />
                    </ListItem>
                ))}
            </Dialog>
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
                        handleVerify={openProviderVerifyDialog}
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
                    onClick={openProviderAddDialog}
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