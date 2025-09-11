// Prompt List Component
function PromptList({ prompts, selectedPrompt, onSelectPrompt, onCreatePrompt, campaignId }) {
    const { useState } = React;
    const [isCreating, setIsCreating] = useState(false);
    const [newPrompt, setNewPrompt] = useState({ title: '', content: '' });

    const handleCreate = async () => {
        if (!newPrompt.title.trim() || !newPrompt.content.trim()) return;
        
        try {
            const prompt = await window.PromptManagerAPI.post('/api/v1/prompts', {
                campaign_id: campaignId,
                title: newPrompt.title,
                content: newPrompt.content,
                variables: []
            });
            onCreatePrompt(prompt);
            setNewPrompt({ title: '', content: '' });
            setIsCreating(false);
        } catch (error) {
            console.error('Failed to create prompt:', error);
        }
    };

    return (
        <div className="bg-white rounded-lg shadow p-4">
            <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold">Prompts</h3>
                {campaignId && (
                    <button
                        onClick={() => setIsCreating(true)}
                        className="bg-blue-500 text-white px-3 py-1 rounded hover:bg-blue-600"
                    >
                        + Add Prompt
                    </button>
                )}
            </div>

            {isCreating && (
                <div className="mb-4 p-3 bg-gray-50 rounded">
                    <input
                        type="text"
                        value={newPrompt.title}
                        onChange={(e) => setNewPrompt({ ...newPrompt, title: e.target.value })}
                        placeholder="Prompt title..."
                        className="w-full px-3 py-2 border rounded mb-2"
                        autoFocus
                    />
                    <textarea
                        value={newPrompt.content}
                        onChange={(e) => setNewPrompt({ ...newPrompt, content: e.target.value })}
                        placeholder="Prompt content..."
                        className="w-full px-3 py-2 border rounded mb-2 h-32"
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
                                setNewPrompt({ title: '', content: '' });
                            }}
                            className="bg-gray-300 px-3 py-1 rounded hover:bg-gray-400"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            <div className="space-y-2 max-h-96 overflow-y-auto">
                {prompts.map(prompt => (
                    <div
                        key={prompt.id}
                        onClick={() => onSelectPrompt(prompt)}
                        className={`p-3 rounded cursor-pointer transition-colors ${
                            selectedPrompt?.id === prompt.id
                                ? 'bg-blue-100 border-blue-500 border'
                                : 'bg-gray-50 hover:bg-gray-100'
                        }`}
                    >
                        <div className="font-medium">{prompt.title}</div>
                        <div className="text-sm text-gray-500 truncate">
                            {prompt.content}
                        </div>
                        <div className="flex items-center gap-4 mt-1">
                            <span className="text-xs text-gray-400">
                                Used {prompt.usage_count || 0} times
                            </span>
                            {prompt.is_favorite && (
                                <span className="text-yellow-500 text-xs">⭐</span>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}