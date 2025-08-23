// Idea Generator - Handles dice roll idea generation and saving

export async function rollDice(state: any): Promise<any> {
    try {
        // Check if we have a campaign selected
        if (!state.current_campaign?.id) {
            showNotification('Please select a campaign first', 'warning');
            return {
                success: false,
                error: 'No campaign selected'
            };
        }
        
        // Show loading animation
        setDiceLoading(true);
        playDiceSound();
        
        // Generate a contextual prompt based on templates
        const prompt = await generateContextualPrompt(state.current_campaign);
        
        // Call idea generation workflow
        const response = await fetch('http://localhost:5678/webhook/generate-ideas', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                campaign_id: state.current_campaign.id,
                prompt: prompt,
                count: 1,
                user_id: state.user_id
            })
        });
        
        const result = await response.json();
        
        if (result.status === 'success' && result.ideas?.length > 0) {
            const idea = result.ideas[0];
            
            // Animate the idea appearing
            await animateIdeaAppearance(idea);
            
            // Update the idea input box
            updateIdeaInput(idea);
            
            // Store temporarily in state
            state.temp_idea = idea;
            
            setDiceLoading(false);
            
            return {
                success: true,
                idea: idea
            };
        } else {
            throw new Error('Failed to generate idea');
        }
    } catch (error) {
        console.error('Error rolling dice:', error);
        setDiceLoading(false);
        showNotification('Failed to generate idea. Please try again.', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

async function generateContextualPrompt(campaign: any): Promise<string> {
    // Load idea templates
    const templates = await loadIdeaTemplates();
    
    // Select random template
    const template = selectRandomTemplate(templates);
    
    // Fill in context variables
    const contextVars = {
        industry: campaign.context?.industry || 'technology',
        target_audience: campaign.context?.target_audience || 'general users',
        domain: campaign.context?.domain || 'business',
        constraints: campaign.context?.constraints?.join(', ') || 'none'
    };
    
    // Replace placeholders in template
    let prompt = template.prompts[Math.floor(Math.random() * template.prompts.length)];
    
    for (const [key, value] of Object.entries(contextVars)) {
        prompt = prompt.replace(new RegExp(`{${key}}`, 'g'), value);
    }
    
    return prompt;
}

export async function saveIdea(state: any): Promise<any> {
    try {
        const ideaText = getIdeaInputValue();
        
        if (!ideaText || ideaText.trim().length === 0) {
            showNotification('Please enter or generate an idea first', 'warning');
            return {
                success: false,
                error: 'No idea to save'
            };
        }
        
        // Parse idea if it's from generation
        let ideaData;
        if (state.temp_idea && state.temp_idea.content === ideaText) {
            ideaData = state.temp_idea;
        } else {
            // User typed their own idea
            ideaData = {
                campaign_id: state.current_campaign.id,
                title: ideaText.substring(0, 100) + (ideaText.length > 100 ? '...' : ''),
                content: ideaText,
                category: 'manual',
                tags: [],
                status: 'draft',
                created_by: state.user_id
            };
        }
        
        // Save to database
        const response = await fetch('http://localhost:5433/api/ideas', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(ideaData)
        });
        
        const savedIdea = await response.json();
        
        // Clear the input
        clearIdeaInput();
        state.temp_idea = null;
        
        // Refresh ideas list
        await refreshIdeasList(state.current_campaign.id);
        
        showNotification('Idea saved successfully!', 'success');
        
        return {
            success: true,
            idea: savedIdea
        };
    } catch (error) {
        console.error('Error saving idea:', error);
        showNotification('Failed to save idea', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

export async function exportIdeas(campaignId: string, format: 'json' | 'csv' | 'markdown' | 'pdf'): Promise<any> {
    try {
        const response = await fetch(`http://localhost:5433/api/campaigns/${campaignId}/export?format=${format}`);
        
        if (!response.ok) {
            throw new Error('Export failed');
        }
        
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const filename = `ideas_${campaignId}_${Date.now()}.${format}`;
        
        // Trigger download
        downloadFile(url, filename);
        
        return {
            success: true,
            filename: filename
        };
    } catch (error) {
        console.error('Error exporting ideas:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

// Helper functions
async function loadIdeaTemplates(): Promise<any> {
    const response = await fetch('/api/config/idea-templates.json');
    return await response.json();
}

function selectRandomTemplate(templates: any): any {
    // Weight-based selection
    const totalWeight = templates.templates.reduce((sum: number, t: any) => sum + (t.weight || 1), 0);
    let random = Math.random() * totalWeight;
    
    for (const template of templates.templates) {
        random -= (template.weight || 1);
        if (random <= 0) {
            return template;
        }
    }
    
    return templates.templates[0];
}

function setDiceLoading(loading: boolean): void {
    // Update UI state
    const diceElement = document.getElementById('dice_button');
    const loadingElement = document.getElementById('dice_loading');
    
    if (diceElement) {
        diceElement.style.opacity = loading ? '0.5' : '1';
        diceElement.style.pointerEvents = loading ? 'none' : 'auto';
    }
    
    if (loadingElement) {
        loadingElement.style.display = loading ? 'block' : 'none';
    }
}

function playDiceSound(): void {
    const audio = new Audio('/sounds/dice-roll.mp3');
    audio.play().catch(e => console.log('Could not play dice sound:', e));
}

async function animateIdeaAppearance(idea: any): Promise<void> {
    // Typewriter effect for idea text
    const inputElement = document.getElementById('idea_input') as HTMLTextAreaElement;
    if (!inputElement) return;
    
    inputElement.value = '';
    const text = idea.content;
    
    for (let i = 0; i < text.length; i++) {
        inputElement.value += text[i];
        await new Promise(resolve => setTimeout(resolve, 20));
    }
}

function updateIdeaInput(idea: any): void {
    const inputElement = document.getElementById('idea_input') as HTMLTextAreaElement;
    if (inputElement) {
        inputElement.value = idea.content;
    }
}

function getIdeaInputValue(): string {
    const inputElement = document.getElementById('idea_input') as HTMLTextAreaElement;
    return inputElement ? inputElement.value : '';
}

function clearIdeaInput(): void {
    const inputElement = document.getElementById('idea_input') as HTMLTextAreaElement;
    if (inputElement) {
        inputElement.value = '';
    }
}

async function refreshIdeasList(campaignId: string): Promise<void> {
    // Trigger UI refresh
    console.log('Refreshing ideas list for campaign:', campaignId);
}

function downloadFile(url: string, filename: string): void {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function showNotification(message: string, type: 'info' | 'success' | 'warning' | 'error'): void {
    console.log(`[${type.toUpperCase()}] ${message}`);
}