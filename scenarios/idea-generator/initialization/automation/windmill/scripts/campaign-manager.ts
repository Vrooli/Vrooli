// Campaign Manager - Handles campaign CRUD operations

export async function switchCampaign(campaignId: string, state: any): Promise<any> {
    try {
        // Fetch campaign details
        const campaign = await fetch(`http://localhost:5433/api/campaigns/${campaignId}`)
            .then(res => res.json());
        
        // Update state with selected campaign
        state.current_campaign = campaign;
        state.current_idea = null;
        state.chat_session = null;
        state.chat_messages = [];
        
        // Refresh ideas list for new campaign
        await refreshIdeasList(campaignId);
        
        return {
            success: true,
            campaign: campaign
        };
    } catch (error) {
        console.error('Error switching campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function createCampaign(data: any, userId: string): Promise<any> {
    try {
        const newCampaign = {
            operation: 'create',
            name: data.name || 'New Campaign',
            description: data.description || '',
            color: data.color || '#3B82F6',
            context: data.context || {},
            user_id: userId
        };
        
        // Call n8n workflow to create campaign
        const response = await fetch('http://localhost:5678/webhook/campaign-sync', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(newCampaign)
        });
        
        const result = await response.json();
        
        if (result.status === 'success') {
            // Refresh campaigns list
            await refreshCampaignsList(userId);
            
            // Switch to new campaign
            await switchCampaign(result.campaign_id, data.state);
        }
        
        return result;
    } catch (error) {
        console.error('Error creating campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function updateCampaign(campaignId: string, updates: any): Promise<any> {
    try {
        const updateRequest = {
            operation: 'update',
            campaign_id: campaignId,
            ...updates
        };
        
        const response = await fetch('http://localhost:5678/webhook/campaign-sync', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(updateRequest)
        });
        
        return await response.json();
    } catch (error) {
        console.error('Error updating campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function deleteCampaign(campaignId: string): Promise<any> {
    try {
        const deleteRequest = {
            operation: 'delete',
            campaign_id: campaignId
        };
        
        const response = await fetch('http://localhost:5678/webhook/campaign-sync', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(deleteRequest)
        });
        
        return await response.json();
    } catch (error) {
        console.error('Error deleting campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function duplicateCampaign(campaignId: string, newName: string, userId: string): Promise<any> {
    try {
        const duplicateRequest = {
            operation: 'duplicate',
            campaign_id: campaignId,
            new_name: newName,
            user_id: userId
        };
        
        const response = await fetch('http://localhost:5678/webhook/campaign-sync', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(duplicateRequest)
        });
        
        return await response.json();
    } catch (error) {
        console.error('Error duplicating campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

async function refreshCampaignsList(userId: string): Promise<void> {
    // This would trigger a re-render of the campaigns tabs
    // Implementation depends on Windmill's state management
    console.log('Refreshing campaigns list for user:', userId);
}

async function refreshIdeasList(campaignId: string): Promise<void> {
    // This would trigger a re-render of the ideas sidebar
    // Implementation depends on Windmill's state management
    console.log('Refreshing ideas list for campaign:', campaignId);
}