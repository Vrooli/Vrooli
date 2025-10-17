// Main App Component
function App() {
    const { useState, useEffect } = React;
    const [campaigns, setCampaigns] = useState([]);
    const [prompts, setPrompts] = useState([]);
    const [selectedCampaign, setSelectedCampaign] = useState(null);
    const [selectedPrompt, setSelectedPrompt] = useState(null);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');

    // Load campaigns
    useEffect(() => {
        const loadCampaigns = async () => {
            try {
                const data = await window.PromptManagerAPI.get('/api/v1/campaigns');
                setCampaigns(data);
            } catch (error) {
                console.error('Failed to load campaigns:', error);
            } finally {
                setLoading(false);
            }
        };
        
        // Wait for API_URL to be set
        setTimeout(loadCampaigns, 500);
    }, []);

    // Load prompts when campaign changes
    useEffect(() => {
        if (!selectedCampaign) {
            setPrompts([]);
            return;
        }

        const loadPrompts = async () => {
            try {
                const data = await window.PromptManagerAPI.get(`/api/v1/campaigns/${selectedCampaign.id}/prompts`);
                setPrompts(data);
            } catch (error) {
                console.error('Failed to load prompts:', error);
            }
        };

        loadPrompts();
    }, [selectedCampaign]);

    const handleSearch = async () => {
        if (!searchQuery.trim()) return;
        
        try {
            const results = await window.PromptManagerAPI.get(`/api/v1/search/prompts?q=${encodeURIComponent(searchQuery)}`);
            setPrompts(results);
            setSelectedCampaign(null);
        } catch (error) {
            console.error('Search failed:', error);
        }
    };

    const handleCreateCampaign = (campaign) => {
        setCampaigns([...campaigns, campaign]);
        setSelectedCampaign(campaign);
    };

    const handleCreatePrompt = (prompt) => {
        setPrompts([...prompts, prompt]);
        setSelectedPrompt(prompt);
    };

    const handleSavePrompt = (updatedPrompt) => {
        setPrompts(prompts.map(p => p.id === updatedPrompt.id ? updatedPrompt : p));
        setSelectedPrompt(updatedPrompt);
    };

    const handleDeletePrompt = (promptId) => {
        setPrompts(prompts.filter(p => p.id !== promptId));
        setSelectedPrompt(null);
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-screen">
                <div className="spinner"></div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            {/* Header */}
            <header className="bg-white shadow-sm border-b">
                <div className="max-w-7xl mx-auto px-4 py-4">
                    <div className="flex items-center justify-between">
                        <h1 className="text-2xl font-bold text-gray-900">
                            📝 Prompt Manager
                        </h1>
                        <div className="flex items-center gap-2">
                            <input
                                type="text"
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                                placeholder="Search prompts..."
                                className="px-3 py-2 border rounded-lg w-64"
                            />
                            <button
                                onClick={handleSearch}
                                className="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600"
                            >
                                Search
                            </button>
                        </div>
                    </div>
                </div>
            </header>

            {/* Main Content */}
            <main className="max-w-7xl mx-auto px-4 py-6">
                <div className="grid grid-cols-12 gap-6">
                    {/* Campaign Tree */}
                    <div className="col-span-3">
                        <CampaignTree
                            campaigns={campaigns}
                            selectedCampaign={selectedCampaign}
                            onSelectCampaign={setSelectedCampaign}
                            onCreateCampaign={handleCreateCampaign}
                        />
                    </div>

                    {/* Prompt List */}
                    <div className="col-span-4">
                        <PromptList
                            prompts={prompts}
                            selectedPrompt={selectedPrompt}
                            onSelectPrompt={setSelectedPrompt}
                            onCreatePrompt={handleCreatePrompt}
                            campaignId={selectedCampaign?.id}
                        />
                    </div>

                    {/* Prompt Editor */}
                    <div className="col-span-5">
                        <PromptEditor
                            prompt={selectedPrompt}
                            onSave={handleSavePrompt}
                            onDelete={handleDeletePrompt}
                        />
                    </div>
                </div>
            </main>
        </div>
    );
}