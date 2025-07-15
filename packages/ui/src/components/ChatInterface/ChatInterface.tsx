import Box from "@mui/material/Box";
import { styled } from "@mui/material/styles";
import { DUMMY_ID, generatePK, type Chat, type ChatParticipantShape, type Session } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getAvailableModels, getExistingAIConfig, getPreferredAvailableModel } from "../../api/ai.js";
import { ChatBubbleTree } from "../ChatBubbleTree/ChatBubbleTree.js";
import { ChatSettingsMenu, type IntegrationSettings, type ModelConfig, type ShareSettings } from "../ChatSettingsMenu/ChatSettingsMenu.js";
import { ModelSelectionErrorBoundary } from "../errors/ModelSelectionErrorBoundary.js";
import { ChatMessageInput } from "../inputs/ChatMessageInput/ChatMessageInput.js";
import { SessionContext } from "../../contexts/session.js";
import { useMessageInput } from "../../hooks/messages.js";
import { useActiveChat } from "../../stores/activeChatStore.js";
import { useModelPreferencesStore } from "../../stores/modelPreferencesStore.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { VALYXA_INFO } from "../../views/objects/chat/ChatCrud.js";
import { ViewDisplayType } from "../../types.js";

export interface ChatInterfaceProps {
    /** Display type for styling */
    display?: ViewDisplayType | `${ViewDisplayType}`;
    /** Whether to show the settings menu button */
    showSettingsButton?: boolean;
    /** Custom placeholder text for the message input */
    placeholder?: string;
    /** Whether the chat interface is disabled */
    disabled?: boolean;
    /** Custom chat ID to use instead of active chat */
    chatId?: string;
    /** Callback when settings menu opens */
    onSettingsOpen?: () => void;
    /** Callback when settings menu closes */
    onSettingsClose?: () => void;
    /** Custom message for when no messages exist */
    noMessagesContent?: React.ReactNode;
    /** Whether to show the interface in compact mode */
    compact?: boolean;
    /** Whether the interface is in a loading state */
    isLoading?: boolean;
    /** Custom loading content */
    loadingContent?: React.ReactNode;
    /** Bot information for single-bot chats */
    bot?: {
        id: string;
        name: string;
        isBot: boolean;
        profileImage?: string;
        updatedAt?: string;
        botSettings?: any;
    } | null;
    /** Whether this is a multi-participant chat */
    isMultiParticipant?: boolean;
}

const ChatInterfaceContainer = styled(Box)<{ compact?: boolean }>(({ theme, compact }) => ({
    display: "flex",
    flexDirection: "column",
    height: compact ? "auto" : "100vh",
    minHeight: compact ? "300px" : "100vh",
    overflow: "hidden",
    backgroundColor: theme.palette.background.default,
}));

const ChatMessagesContainer = styled(Box)(({ theme }) => ({
    flex: 1,
    overflowY: "auto",
    overflowX: "hidden",
    display: "flex",
    flexDirection: "column",
    backgroundColor: theme.palette.background.default,
    position: "relative",
    // Add padding to prevent content cutoff
    paddingTop: theme.spacing(2),
    paddingBottom: theme.spacing(4), // Extra space at bottom
}));

const EmptyStateWrapper = styled(Box)({
    flex: 1,
    display: "flex",
    flexDirection: "column",
});

const MESSAGE_LIST_ID = "chatInterface";

export function ChatInterface({
    display = ViewDisplayType.Page,
    showSettingsButton = true,
    placeholder,
    disabled = false,
    chatId,
    onSettingsOpen,
    onSettingsClose,
    noMessagesContent,
    compact = false,
    isLoading = false,
    loadingContent,
}: ChatInterfaceProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // State for the ChatSettingsMenu
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);

    // Get available models for default config
    const availableModels = useMemo(() => {
        const config = getExistingAIConfig()?.service?.config;
        return config ? getAvailableModels(config) : [];
    }, []);

    // Default model config state - will be initialized after availableModels loads
    const [modelConfigs, setModelConfigs] = useState<ModelConfig[]>([]);
    const [isModelConfigsInitialized, setIsModelConfigsInitialized] = useState(false);

    // Participants for dashboard context (e.g., just Valyxa)
    const [participants, setParticipants] = useState<ChatParticipantShape[]>([
        {
            __typename: "ChatParticipant",
            id: generatePK(),
            user: { ...VALYXA_INFO, __typename: "User" },
        } as ChatParticipantShape,
    ]);

    // Default share settings for dashboard
    const [shareSettings, setShareSettings] = useState<ShareSettings>({ enabled: false });

    // Default integration settings for dashboard
    const [integrationSettings, setIntegrationSettings] = useState<IntegrationSettings>({ projects: [], externalApps: [] });

    // Placeholder chat object for the menu props
    const settingsChat = useMemo(() => ({
        __typename: "Chat" as const,
        id: chatId || DUMMY_ID,
        publicId: chatId || DUMMY_ID,
        translations: [{
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "Chat Settings",
            description: "Configure chat settings and model preferences.",
        }],
    }), [session, chatId]);

    // Initialize model configs when available models are loaded
    useEffect(() => {
        if (!isModelConfigsInitialized) {
            if (availableModels.length > 0) {
                try {
                    const preferredModel = getPreferredAvailableModel(availableModels);
                    const modelToUse = preferredModel || availableModels[0];
                    
                    setModelConfigs([{
                        model: modelToUse,
                        toolSettings: { siteActions: true, webSearch: true, fileRetrieval: true },
                        requireConfirmation: false,
                    }]);
                    setIsModelConfigsInitialized(true);
                } catch (error) {
                    // Fallback to first available model without preferences
                    if (availableModels.length > 0) {
                        setModelConfigs([{
                            model: availableModels[0],
                            toolSettings: { siteActions: true, webSearch: true, fileRetrieval: true },
                            requireConfirmation: false,
                        }]);
                        setIsModelConfigsInitialized(true);
                    }
                }
            } else {
                // No models available - set as initialized to prevent infinite loop
                setIsModelConfigsInitialized(true);
            }
        }
    }, [availableModels, isModelConfigsInitialized]);

    // Update local state when model config changes
    const handleModelConfigChange = useCallback((newConfigs: ModelConfig[]) => {
        setModelConfigs(newConfigs);
        
        // Save the model preference if a model is selected
        if (newConfigs.length > 0 && newConfigs[0].model) {
            useModelPreferencesStore.getState().setPreferredModel(newConfigs[0].model.value);
        }
    }, []);

    // Settings menu handlers
    const handleOpenSettings = useCallback(() => {
        setIsSettingsOpen(true);
        onSettingsOpen?.();
    }, [onSettingsOpen]);

    const handleCloseSettings = useCallback(() => {
        setIsSettingsOpen(false);
        onSettingsClose?.();
    }, [onSettingsClose]);

    // Placeholder callbacks for settings menu
    const handleAddParticipant = useCallback((id: string) => { 
        console.log("Add participant:", id);
    }, []);
    
    const handleRemoveParticipant = useCallback((id: string) => { 
        console.log("Remove participant:", id);
    }, []);
    
    const handleUpdateDetails = useCallback((data: { name: string; description?: string }) => { 
        console.log("Update details:", data);
    }, []);
    
    const handleToggleShare = useCallback((enabled: boolean) => {
        // Generate invite link when enabled, clear on disable
        const link = enabled ? `${window.location.origin}/chat/${settingsChat.publicId}` : undefined;
        setShareSettings({ enabled, link });
    }, [settingsChat.publicId]);
    
    const handleIntegrationSettingsChange = useCallback((newSettings: IntegrationSettings) => {
        setIntegrationSettings(newSettings);
    }, []);
    
    // Placeholder for delete chat - not applicable for general chat interface
    const handleDeleteChat = useCallback(() => {
        console.log("Delete chat");
    }, []);

    // Chat state and message handling
    const [message, setMessage] = useState<string>("");
    
    const {
        chat,
        isBotOnlyChat,
        isLoading: isChatLoading,
        messageActions,
        messageStream,
        messageTree,
        usersTyping: participantsTyping,
        resetActiveChat,
        taskInfo,
    } = useActiveChat({ setMessage });

    const messageInput = useMessageInput({
        id: MESSAGE_LIST_ID,
        languages: getUserLanguages(session),
        message,
        postMessage: messageActions.postMessage,
        putMessage: messageActions.putMessage,
        replyToMessage: messageActions.replyToMessage,
        setMessage,
    });

    // Ensure taskInfo has contexts property for safety
    const safeTaskInfo = useMemo(() => ({
        ...taskInfo,
        contexts: taskInfo.contexts || {},
    }), [taskInfo]);

    const hasMessages = useMemo(() => {
        return messageTree.tree.getMessagesCount() > 0;
    }, [messageTree]);

    const defaultPlaceholder = placeholder || t("WhatWouldYouLikeToDo", { defaultValue: "What would you like to do?" });

    return (
        <ChatInterfaceContainer compact={compact}>
            <ChatMessagesContainer>
                {isLoading && (
                    <EmptyStateWrapper>
                        {loadingContent}
                    </EmptyStateWrapper>
                )}
                {!isLoading && !hasMessages && (
                    <EmptyStateWrapper>
                        {noMessagesContent}
                    </EmptyStateWrapper>
                )}
                {!isLoading && hasMessages && (
                    <ChatBubbleTree
                        branches={messageTree.branches}
                        handleEdit={messageInput.startEditingMessage}
                        handleRegenerateResponse={messageActions.regenerateResponse}
                        handleReply={messageInput.startReplyingToMessage}
                        handleRetry={messageActions.retryPostMessage}
                        isBotOnlyChat={isBotOnlyChat}
                        isEditingMessage={Boolean(messageInput.messageBeingEdited)}
                        isReplyingToMessage={Boolean(messageInput.messageBeingRepliedTo)}
                        messageStream={messageStream}
                        removeMessages={messageTree.removeMessages}
                        setBranches={messageTree.setBranches}
                        tree={messageTree.tree}
                    />
                )}
            </ChatMessagesContainer>
            
            <ChatMessageInput
                disabled={disabled}
                display={display}
                isLoading={isChatLoading}
                message={message}
                messageBeingEdited={messageInput.messageBeingEdited}
                messageBeingRepliedTo={messageInput.messageBeingRepliedTo}
                participantsTyping={participantsTyping}
                placeholder={defaultPlaceholder}
                setMessage={setMessage}
                stopEditingMessage={messageInput.stopEditingMessage}
                stopReplyingToMessage={messageInput.stopReplyingToMessage}
                submitMessage={messageInput.submitMessage}
                taskInfo={safeTaskInfo}
            />

            {/* Render the settings menu dialog */}
            {showSettingsButton && (
                <ModelSelectionErrorBoundary onRetry={() => {}}>
                    <ChatSettingsMenu
                        open={isSettingsOpen}
                        onClose={handleCloseSettings}
                        chat={settingsChat}
                        participants={participants}
                        modelConfigs={modelConfigs}
                        shareSettings={shareSettings}
                        integrationSettings={integrationSettings}
                        onModelConfigChange={handleModelConfigChange}
                        onAddParticipant={handleAddParticipant}
                        onRemoveParticipant={handleRemoveParticipant}
                        onUpdateDetails={handleUpdateDetails}
                        onToggleShare={handleToggleShare}
                        onIntegrationSettingsChange={handleIntegrationSettingsChange}
                        onDeleteChat={handleDeleteChat}
                    />
                </ModelSelectionErrorBoundary>
            )}
        </ChatInterfaceContainer>
    );
}
