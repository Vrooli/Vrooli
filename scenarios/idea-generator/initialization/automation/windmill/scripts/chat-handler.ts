// Chat Handler - Manages real-time chat for idea refinement

let websocket: WebSocket | null = null;

export async function openIdeaChat(ideaId: string, state: any): Promise<any> {
    try {
        // Fetch idea details
        const response = await fetch(`http://localhost:5433/api/ideas/${ideaId}`);
        const idea = await response.json();
        
        // Set current idea in state
        state.current_idea = idea;
        
        // Check for existing chat session
        const sessionResponse = await fetch(`http://localhost:5433/api/chat/sessions?idea_id=${ideaId}&status=active`);
        const sessions = await sessionResponse.json();
        
        if (sessions.length > 0) {
            state.chat_session = sessions[0];
            await loadChatHistory(sessions[0].id, state);
        } else {
            // Create new session
            const newSession = await createChatSession(ideaId, state.user_id);
            state.chat_session = newSession;
        }
        
        // Open chat panel
        showChatPanel(true);
        
        // Initialize WebSocket connection
        await initializeWebSocket(state);
        
        return {
            success: true,
            session: state.chat_session
        };
    } catch (error) {
        console.error('Error opening idea chat:', error);
        showNotification('Failed to open chat', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

export async function startRefinement(state: any): Promise<any> {
    try {
        // Get the idea text from input
        const ideaText = getIdeaInputValue();
        
        if (!ideaText || ideaText.trim().length === 0) {
            showNotification('Please enter or generate an idea first', 'warning');
            return {
                success: false,
                error: 'No idea to refine'
            };
        }
        
        // Save idea first if it's not saved
        let ideaId = state.current_idea?.id;
        
        if (!ideaId) {
            const saveResult = await saveIdeaForRefinement(ideaText, state);
            if (!saveResult.success) {
                return saveResult;
            }
            ideaId = saveResult.idea.id;
        }
        
        // Open chat with the idea
        return await openIdeaChat(ideaId, state);
    } catch (error) {
        console.error('Error starting refinement:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function sendMessage(state: any): Promise<any> {
    try {
        const messageInput = document.getElementById('chat_input') as HTMLTextAreaElement;
        const message = messageInput?.value?.trim();
        
        if (!message) {
            return {
                success: false,
                error: 'Message is empty'
            };
        }
        
        if (!state.chat_session?.id) {
            showNotification('No active chat session', 'error');
            return {
                success: false,
                error: 'No active session'
            };
        }
        
        // Add user message to UI immediately
        addMessageToUI({
            sender_type: 'user',
            sender_name: state.user_name || 'You',
            message_text: message,
            created_at: new Date().toISOString()
        }, state);
        
        // Clear input
        if (messageInput) {
            messageInput.value = '';
        }
        
        // Show typing indicator
        showTypingIndicator(true);
        
        // Send to chat refinement workflow
        const response = await fetch('http://localhost:5678/webhook/chat-message', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                session_id: state.chat_session.id,
                idea_id: state.current_idea.id,
                user_id: state.user_id,
                message: message,
                agent_type: state.selected_agent || 'auto',
                include_context: true,
                include_documents: true
            })
        });
        
        const result = await response.json();
        
        if (result.status === 'success') {
            // Add agent response to UI
            addMessageToUI({
                sender_type: 'agent',
                sender_name: result.agent_type,
                message_text: result.response,
                created_at: result.timestamp
            }, state);
            
            // Hide typing indicator
            showTypingIndicator(false);
            
            return {
                success: true,
                response: result.response
            };
        } else {
            throw new Error(result.message || 'Failed to get response');
        }
    } catch (error) {
        console.error('Error sending message:', error);
        showTypingIndicator(false);
        showNotification('Failed to send message', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

export async function closeChat(state: any): Promise<void> {
    // Close WebSocket connection
    if (websocket) {
        websocket.close();
        websocket = null;
    }
    
    // Hide chat panel
    showChatPanel(false);
    
    // Clear chat state
    state.chat_session = null;
    state.chat_messages = [];
    state.current_idea = null;
}

export async function storeIdea(state: any): Promise<any> {
    try {
        if (!state.current_idea?.id) {
            showNotification('No idea to store', 'warning');
            return {
                success: false,
                error: 'No current idea'
            };
        }
        
        // Update idea status to finalized
        const response = await fetch(`http://localhost:5433/api/ideas/${state.current_idea.id}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                status: 'finalized',
                metadata: {
                    ...state.current_idea.metadata,
                    finalized_at: new Date().toISOString(),
                    chat_session_id: state.chat_session?.id
                }
            })
        });
        
        const updatedIdea = await response.json();
        
        // Update state
        state.current_idea = updatedIdea;
        
        // Refresh ideas list
        await refreshIdeasList(state.current_campaign?.id);
        
        showNotification('Idea stored successfully!', 'success');
        
        return {
            success: true,
            idea: updatedIdea
        };
    } catch (error) {
        console.error('Error storing idea:', error);
        showNotification('Failed to store idea', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

// Helper functions
async function createChatSession(ideaId: string, userId: string): Promise<any> {
    const response = await fetch('http://localhost:5433/api/chat/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            idea_id: ideaId,
            user_id: userId,
            session_type: 'refinement',
            status: 'active'
        })
    });
    
    return await response.json();
}

async function loadChatHistory(sessionId: string, state: any): Promise<void> {
    const response = await fetch(`http://localhost:5433/api/chat/messages?session_id=${sessionId}`);
    const messages = await response.json();
    
    state.chat_messages = messages;
    
    // Update UI with messages
    messages.forEach((msg: any) => addMessageToUI(msg, state));
}

async function initializeWebSocket(state: any): Promise<void> {
    if (websocket) {
        websocket.close();
    }
    
    websocket = new WebSocket(`ws://localhost:5683/ws/chat?session=${state.chat_session.id}`);
    
    websocket.onopen = () => {
        console.log('WebSocket connected');
    };
    
    websocket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        
        if (data.type === 'message') {
            addMessageToUI(data.message, state);
        } else if (data.type === 'typing') {
            showTypingIndicator(data.isTyping);
        }
    };
    
    websocket.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    websocket.onclose = () => {
        console.log('WebSocket disconnected');
    };
}

function addMessageToUI(message: any, state: any): void {
    // Add to state
    state.chat_messages.push(message);
    
    // Get agent info for styling
    const agentInfo = getAgentInfo(message.sender_name);
    
    // Create message element
    const messageContainer = document.createElement('div');
    messageContainer.className = `chat-message ${message.sender_type}`;
    
    // Add avatar
    const avatar = document.createElement('div');
    avatar.className = 'message-avatar';
    avatar.textContent = message.sender_type === 'user' ? '👤' : agentInfo.avatar;
    avatar.style.backgroundColor = message.sender_type === 'user' ? '#3B82F6' : agentInfo.color;
    
    // Add content
    const content = document.createElement('div');
    content.className = 'message-content';
    
    const sender = document.createElement('div');
    sender.className = 'message-sender';
    sender.textContent = message.sender_type === 'user' ? 'You' : agentInfo.name;
    
    const text = document.createElement('div');
    text.className = 'message-text';
    text.textContent = message.message_text;
    
    const timestamp = document.createElement('div');
    timestamp.className = 'message-timestamp';
    timestamp.textContent = formatTimestamp(message.created_at);
    
    content.appendChild(sender);
    content.appendChild(text);
    content.appendChild(timestamp);
    
    messageContainer.appendChild(avatar);
    messageContainer.appendChild(content);
    
    // Add to messages list
    const messagesList = document.getElementById('messages_list');
    if (messagesList) {
        messagesList.appendChild(messageContainer);
        messagesList.scrollTop = messagesList.scrollHeight;
    }
}

function getAgentInfo(agentType: string): any {
    const agents: Record<string, any> = {
        revise: { name: 'Revision Agent', avatar: '🔧', color: '#3B82F6' },
        investigate: { name: 'Research Agent', avatar: '🔍', color: '#10B981' },
        critique: { name: 'Critique Agent', avatar: '⚖️', color: '#F59E0B' },
        expand: { name: 'Creative Agent', avatar: '💡', color: '#8B5CF6' },
        synthesize: { name: 'Synthesis Agent', avatar: '🔮', color: '#EC4899' },
        validate: { name: 'Validator Agent', avatar: '✅', color: '#059669' }
    };
    
    return agents[agentType] || { name: 'AI Agent', avatar: '🤖', color: '#6B7280' };
}

function showChatPanel(show: boolean): void {
    const chatPanel = document.getElementById('chat_panel');
    if (chatPanel) {
        chatPanel.style.display = show ? 'flex' : 'none';
    }
}

function showTypingIndicator(show: boolean): void {
    const indicator = document.getElementById('typing_indicator');
    if (indicator) {
        indicator.style.display = show ? 'block' : 'none';
    }
}

function formatTimestamp(timestamp: string): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    if (diff < 60000) {
        return 'Just now';
    } else if (diff < 3600000) {
        return `${Math.floor(diff / 60000)} min ago`;
    } else if (diff < 86400000) {
        return `${Math.floor(diff / 3600000)} hr ago`;
    } else {
        return date.toLocaleDateString();
    }
}

async function saveIdeaForRefinement(ideaText: string, state: any): Promise<any> {
    const response = await fetch('http://localhost:5433/api/ideas', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            campaign_id: state.current_campaign.id,
            title: ideaText.substring(0, 100) + (ideaText.length > 100 ? '...' : ''),
            content: ideaText,
            category: 'refinement',
            status: 'draft',
            created_by: state.user_id
        })
    });
    
    const idea = await response.json();
    
    return {
        success: true,
        idea: idea
    };
}

function getIdeaInputValue(): string {
    const inputElement = document.getElementById('idea_input') as HTMLTextAreaElement;
    return inputElement ? inputElement.value : '';
}

async function refreshIdeasList(campaignId: string | null): Promise<void> {
    console.log('Refreshing ideas list for campaign:', campaignId);
}

function showNotification(message: string, type: 'info' | 'success' | 'warning' | 'error'): void {
    console.log(`[${type.toUpperCase()}] ${message}`);
}