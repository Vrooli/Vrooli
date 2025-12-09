import { useCallback, useMemo, useState } from 'react';
import { useSessionProfiles } from '@features/record-mode/hooks/useSessionProfiles';
import type { RecordingSessionProfile } from '@features/record-mode/types';

const formatLastUsed = (value: string) => {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return 'Unknown';
  }
  return parsed.toLocaleString();
};

export function SessionProfilesTab() {
  const { profiles, loading, error, create, rename, remove, refresh } = useSessionProfiles();
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [nameDraft, setNameDraft] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const sortedProfiles = useMemo(
    () =>
      [...profiles].sort((a, b) => new Date(b.lastUsedAt).getTime() - new Date(a.lastUsedAt).getTime()),
    [profiles]
  );

  const handleStartRename = useCallback((profile: RecordingSessionProfile) => {
    setRenamingId(profile.id);
    setNameDraft(profile.name);
  }, []);

  const handleRenameSave = useCallback(async () => {
    if (!renamingId || !nameDraft.trim()) return;
    await rename(renamingId, nameDraft.trim());
    setRenamingId(null);
    setNameDraft('');
  }, [nameDraft, rename, renamingId]);

  const handleDelete = useCallback(
    async (id: string) => {
      const confirmed = window.confirm('Delete this session? This will remove saved login state.');
      if (!confirmed) return;
      setDeletingId(id);
      await remove(id);
      setDeletingId(null);
    },
    [remove]
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Playwright sessions</h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Reuse saved sessions to stay signed in while recording and replaying workflows.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-60"
            onClick={() => void create()}
            disabled={loading}
          >
            New session
          </button>
          <button
            className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700"
            onClick={() => void refresh()}
            disabled={loading}
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/40 dark:text-red-200">
          {error}
        </div>
      )}

      <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-700">
          <span className="text-sm text-gray-700 dark:text-gray-200">
            {loading ? 'Loading sessions…' : `${profiles.length} saved session${profiles.length === 1 ? '' : 's'}`}
          </span>
        </div>

        {sortedProfiles.length === 0 && !loading ? (
          <div className="px-4 py-6 text-sm text-gray-600 dark:text-gray-300">
            No saved sessions yet. Start recording to create one automatically.
          </div>
        ) : (
          <ul className="divide-y divide-gray-200 dark:divide-gray-800">
            {sortedProfiles.map((profile) => (
              <li key={profile.id} className="px-4 py-3 flex items-center justify-between gap-3">
                <div className="min-w-0">
                  {renamingId === profile.id ? (
                    <div className="flex items-center gap-2">
                      <input
                        className="w-48 rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-2 py-1 text-sm text-gray-900 dark:text-gray-100"
                        value={nameDraft}
                        onChange={(e) => setNameDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            void handleRenameSave();
                          }
                        }}
                      />
                      <button
                        className="px-2 py-1 text-xs font-medium text-white bg-green-600 rounded-md hover:bg-green-700 disabled:opacity-60"
                        onClick={() => void handleRenameSave()}
                        disabled={!nameDraft.trim()}
                      >
                        Save
                      </button>
                      <button
                        className="px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700"
                        onClick={() => {
                          setRenamingId(null);
                          setNameDraft('');
                        }}
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <>
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate">{profile.name}</p>
                        {profile.hasStorageState && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-200">
                            Auth saved
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Last used {formatLastUsed(profile.lastUsedAt)}
                      </p>
                    </>
                  )}
                </div>

                <div className="flex items-center gap-2">
                  {renamingId !== profile.id && (
                    <button
                      className="px-2 py-1 text-xs font-medium text-blue-700 dark:text-blue-200 bg-blue-50 dark:bg-blue-900/40 rounded-md hover:bg-blue-100 dark:hover:bg-blue-900/60"
                      onClick={() => handleStartRename(profile)}
                    >
                      Rename
                    </button>
                  )}
                  <button
                    className="px-2 py-1 text-xs font-medium text-red-700 dark:text-red-200 bg-red-50 dark:bg-red-900/40 rounded-md hover:bg-red-100 dark:hover:bg-red-900/60 disabled:opacity-60"
                    onClick={() => void handleDelete(profile.id)}
                    disabled={deletingId === profile.id}
                  >
                    {deletingId === profile.id ? 'Deleting…' : 'Delete'}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
