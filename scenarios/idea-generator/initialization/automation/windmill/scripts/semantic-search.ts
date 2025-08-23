// Semantic Search - Handles search functionality across ideas and documents

export async function searchIdeas(query: string, state: any): Promise<any> {
    try {
        // Don't search if query is too short
        if (query.trim().length < 2) {
            return {
                success: true,
                results: []
            };
        }
        
        // Build search request
        const searchRequest = {
            query: query,
            campaign_id: state.current_campaign?.id || null,
            limit: 20,
            content_types: ['idea', 'document'],
            filters: {
                status: state.filter_status !== 'All' ? state.filter_status.toLowerCase() : null
            },
            user_id: state.user_id,
            include_similarity_scores: true,
            min_similarity: 0.7,
            boost_recent: true
        };
        
        // Call semantic search workflow
        const response = await fetch('http://localhost:5678/webhook/semantic-search', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(searchRequest)
        });
        
        const result = await response.json();
        
        if (result.status === 'success') {
            // Update UI with search results
            updateSearchResults(result.results);
            
            // Update state
            state.search_results = result.results;
            state.search_query = query;
            
            // Show result count
            showSearchResultCount(result.total_found);
            
            return {
                success: true,
                results: result.results,
                total: result.total_found
            };
        } else {
            throw new Error(result.message || 'Search failed');
        }
    } catch (error) {
        console.error('Error searching ideas:', error);
        showNotification('Search failed. Please try again.', 'error');
        
        return {
            success: false,
            error: error.message
        };
    }
}

export async function clearSearch(state: any): Promise<void> {
    state.search_query = '';
    state.search_results = [];
    
    // Reset ideas list to show all
    await refreshIdeasList(state.current_campaign?.id);
}

export async function searchByCampaign(campaignId: string): Promise<any> {
    try {
        const response = await fetch(`http://localhost:5433/api/ideas?campaign_id=${campaignId}`);
        const ideas = await response.json();
        
        return {
            success: true,
            results: ideas
        };
    } catch (error) {
        console.error('Error searching by campaign:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function searchByTags(tags: string[], state: any): Promise<any> {
    try {
        const query = tags.join(' OR ');
        return await searchIdeas(query, state);
    } catch (error) {
        console.error('Error searching by tags:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function searchSimilar(ideaId: string): Promise<any> {
    try {
        // Get idea embedding
        const ideaResponse = await fetch(`http://localhost:5433/api/ideas/${ideaId}`);
        const idea = await ideaResponse.json();
        
        if (!idea.embedding) {
            throw new Error('Idea has no embedding for similarity search');
        }
        
        // Search for similar ideas using Qdrant
        const searchResponse = await fetch('http://localhost:6333/collections/ideas/points/search', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                vector: idea.embedding,
                limit: 10,
                score_threshold: 0.7,
                with_payload: true,
                filter: {
                    must_not: [
                        {
                            key: 'idea_id',
                            match: { value: ideaId }
                        }
                    ]
                }
            })
        });
        
        const results = await searchResponse.json();
        
        return {
            success: true,
            results: results.result || []
        };
    } catch (error) {
        console.error('Error finding similar ideas:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function getSearchSuggestions(query: string): Promise<string[]> {
    try {
        // Get recent search queries for suggestions
        const response = await fetch(`http://localhost:5433/api/search/suggestions?q=${encodeURIComponent(query)}`);
        const suggestions = await response.json();
        
        return suggestions.slice(0, 8);
    } catch (error) {
        console.error('Error getting search suggestions:', error);
        return [];
    }
}

export async function trackSearchClick(resultId: string, resultType: 'idea' | 'document', queryId: string): Promise<void> {
    try {
        // Track which results users click on for improving search
        await fetch('http://localhost:5433/api/search/track-click', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                query_id: queryId,
                result_id: resultId,
                result_type: resultType,
                timestamp: new Date().toISOString()
            })
        });
    } catch (error) {
        console.error('Error tracking search click:', error);
    }
}

// Helper functions
function updateSearchResults(results: any[]): void {
    const listElement = document.getElementById('ideas_list');
    if (!listElement) return;
    
    // Clear existing items
    listElement.innerHTML = '';
    
    // Add search results
    results.forEach(result => {
        const itemElement = createSearchResultElement(result);
        listElement.appendChild(itemElement);
    });
}

function createSearchResultElement(result: any): HTMLElement {
    const div = document.createElement('div');
    div.className = 'search-result-item';
    div.dataset.id = result.id;
    div.dataset.type = result.type;
    
    // Add type indicator
    const typeIndicator = document.createElement('span');
    typeIndicator.className = 'type-indicator';
    typeIndicator.textContent = result.type === 'idea' ? '💡' : '📄';
    
    // Add title
    const title = document.createElement('h4');
    title.textContent = result.title;
    title.className = 'result-title';
    
    // Add content preview
    const content = document.createElement('p');
    content.textContent = result.content;
    content.className = 'result-content';
    
    // Add similarity score if available
    if (result.score) {
        const score = document.createElement('span');
        score.className = 'similarity-score';
        score.textContent = `${Math.round(result.score * 100)}% match`;
        div.appendChild(score);
    }
    
    // Add campaign badge
    if (result.campaign_name) {
        const badge = document.createElement('span');
        badge.className = 'campaign-badge';
        badge.style.backgroundColor = result.campaign_color || '#3B82F6';
        badge.textContent = result.campaign_name;
        div.appendChild(badge);
    }
    
    div.appendChild(typeIndicator);
    div.appendChild(title);
    div.appendChild(content);
    
    // Add click handler
    div.addEventListener('click', () => {
        if (result.type === 'idea') {
            openIdeaDetails(result.id);
        } else {
            openDocumentDetails(result.id);
        }
    });
    
    return div;
}

function showSearchResultCount(count: number): void {
    const countElement = document.getElementById('search_result_count');
    if (countElement) {
        countElement.textContent = `Found ${count} result${count !== 1 ? 's' : ''}`;
    }
}

async function refreshIdeasList(campaignId: string | null): Promise<void> {
    console.log('Refreshing ideas list for campaign:', campaignId);
}

function openIdeaDetails(ideaId: string): void {
    console.log('Opening idea details:', ideaId);
}

function openDocumentDetails(documentId: string): void {
    console.log('Opening document details:', documentId);
}

function showNotification(message: string, type: 'error'): void {
    console.error(message);
}