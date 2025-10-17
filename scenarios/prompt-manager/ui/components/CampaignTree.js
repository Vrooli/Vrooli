// Campaign Tree Component
function CampaignTree({ campaigns, selectedCampaign, onSelectCampaign, onCreateCampaign }) {
    const { useState } = React;
    const [isCreating, setIsCreating] = useState(false);
    const [newCampaignName, setNewCampaignName] = useState('');

    const handleCreate = async () => {
        if (!newCampaignName.trim()) return;
        
        try {
            const campaign = await window.PromptManagerAPI.post('/api/v1/campaigns', {
                name: newCampaignName,
                description: '',
                color: '#6366f1',
                icon: 'folder'
            });
            onCreateCampaign(campaign);
            setNewCampaignName('');
            setIsCreating(false);
        } catch (error) {
            console.error('Failed to create campaign:', error);
        }
    };

    return (
        <div className="bg-white rounded-lg shadow p-4">
            <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold">Campaigns</h3>
                <button
                    onClick={() => setIsCreating(true)}
                    className="bg-blue-500 text-white px-3 py-1 rounded hover:bg-blue-600"
                >
                    + New
                </button>
            </div>
            
            {isCreating && (
                <div className="mb-4 p-3 bg-gray-50 rounded">
                    <input
                        type="text"
                        value={newCampaignName}
                        onChange={(e) => setNewCampaignName(e.target.value)}
                        placeholder="Campaign name..."
                        className="w-full px-3 py-2 border rounded mb-2"
                        autoFocus
                    />
                    <div className="flex gap-2">
                        <button
                            onClick={handleCreate}
                            className="bg-green-500 text-white px-3 py-1 rounded hover:bg-green-600"
                        >
                            Create
                        </button>
                        <button
                            onClick={() => {
                                setIsCreating(false);
                                setNewCampaignName('');
                            }}
                            className="bg-gray-300 px-3 py-1 rounded hover:bg-gray-400"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            <div className="space-y-2">
                {campaigns.map(campaign => (
                    <div
                        key={campaign.id}
                        onClick={() => onSelectCampaign(campaign)}
                        className={`p-3 rounded cursor-pointer transition-colors ${
                            selectedCampaign?.id === campaign.id
                                ? 'bg-blue-100 border-blue-500 border'
                                : 'bg-gray-50 hover:bg-gray-100'
                        }`}
                    >
                        <div className="flex items-center justify-between">
                            <div>
                                <div className="font-medium">{campaign.name}</div>
                                <div className="text-sm text-gray-500">
                                    {campaign.prompt_count || 0} prompts
                                </div>
                            </div>
                            {campaign.is_favorite && (
                                <span className="text-yellow-500">⭐</span>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}