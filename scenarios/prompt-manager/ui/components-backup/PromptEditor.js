// Prompt Editor Component
function PromptEditor({ prompt, onSave, onDelete }) {
    const { useState, useEffect } = React;
    const [editedPrompt, setEditedPrompt] = useState(prompt || {});
    const [isSaving, setIsSaving] = useState(false);

    useEffect(() => {
        setEditedPrompt(prompt || {});
    }, [prompt]);

    const handleSave = async () => {
        if (!editedPrompt.id) return;
        
        setIsSaving(true);
        try {
            const updated = await window.PromptManagerAPI.put(`/api/v1/prompts/${editedPrompt.id}`, {
                title: editedPrompt.title,
                content: editedPrompt.content,
                description: editedPrompt.description,
                is_favorite: editedPrompt.is_favorite
            });
            onSave(updated);
        } catch (error) {
            console.error('Failed to save prompt:', error);
        } finally {
            setIsSaving(false);
        }
    };

    const handleDelete = async () => {
        if (!editedPrompt.id || !confirm('Delete this prompt?')) return;
        
        try {
            await window.PromptManagerAPI.delete(`/api/v1/prompts/${editedPrompt.id}`);
            onDelete(editedPrompt.id);
        } catch (error) {
            console.error('Failed to delete prompt:', error);
        }
    };

    const handleTest = async () => {
        if (!editedPrompt.id) return;
        
        try {
            const result = await window.PromptManagerAPI.post(`/api/v1/prompts/${editedPrompt.id}/test`, {
                model: 'llama3.2',
                variables: {}
            });
            alert(`Test Result:\n\n${result.response || 'No response'}`);
        } catch (error) {
            console.error('Failed to test prompt:', error);
            alert('Testing is not available (Ollama may not be configured)');
        }
    };

    if (!prompt) {
        return (
            <div className="bg-white rounded-lg shadow p-8 text-center text-gray-500">
                Select a prompt to edit
            </div>
        );
    }

    return (
        <div className="bg-white rounded-lg shadow p-4">
            <div className="flex justify-between items-start mb-4">
                <h3 className="text-lg font-semibold">Edit Prompt</h3>
                <div className="flex gap-2">
                    <button
                        onClick={handleTest}
                        className="bg-purple-500 text-white px-3 py-1 rounded hover:bg-purple-600"
                    >
                        Test
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={isSaving}
                        className="bg-green-500 text-white px-3 py-1 rounded hover:bg-green-600 disabled:opacity-50"
                    >
                        {isSaving ? 'Saving...' : 'Save'}
                    </button>
                    <button
                        onClick={handleDelete}
                        className="bg-red-500 text-white px-3 py-1 rounded hover:bg-red-600"
                    >
                        Delete
                    </button>
                </div>
            </div>

            <div className="space-y-4">
                <div>
                    <label className="block text-sm font-medium mb-1">Title</label>
                    <input
                        type="text"
                        value={editedPrompt.title || ''}
                        onChange={(e) => setEditedPrompt({ ...editedPrompt, title: e.target.value })}
                        className="w-full px-3 py-2 border rounded"
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-1">Content</label>
                    <textarea
                        value={editedPrompt.content || ''}
                        onChange={(e) => setEditedPrompt({ ...editedPrompt, content: e.target.value })}
                        className="w-full px-3 py-2 border rounded h-64 font-mono text-sm"
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-1">Description</label>
                    <textarea
                        value={editedPrompt.description || ''}
                        onChange={(e) => setEditedPrompt({ ...editedPrompt, description: e.target.value })}
                        className="w-full px-3 py-2 border rounded h-20"
                    />
                </div>

                <div>
                    <label className="flex items-center gap-2">
                        <input
                            type="checkbox"
                            checked={editedPrompt.is_favorite || false}
                            onChange={(e) => setEditedPrompt({ ...editedPrompt, is_favorite: e.target.checked })}
                        />
                        <span>Favorite</span>
                    </label>
                </div>

                <div className="pt-4 border-t text-sm text-gray-500">
                    <div>Created: {new Date(editedPrompt.created_at).toLocaleString()}</div>
                    <div>Updated: {new Date(editedPrompt.updated_at).toLocaleString()}</div>
                    <div>Usage Count: {editedPrompt.usage_count || 0}</div>
                    {editedPrompt.word_count && (
                        <div>Word Count: {editedPrompt.word_count}</div>
                    )}
                    {editedPrompt.estimated_tokens && (
                        <div>Estimated Tokens: {editedPrompt.estimated_tokens}</div>
                    )}
                </div>
            </div>
        </div>
    );
}