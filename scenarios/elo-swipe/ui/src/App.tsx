import { useCallback, useEffect, useState } from 'react';
import { TopBar } from './components/TopBar';
import { ListPicker } from './components/ListPicker';
import { StatusStrip } from './components/StatusStrip';
import { InstructionPanel } from './components/InstructionPanel';
import { ComparisonStage } from './components/ComparisonStage';
import { Modal } from './components/Modal';
import { RankingsPanel } from './components/RankingsPanel';
import { CreateListForm } from './components/CreateListForm';
import { apiFetch, API_BASE_URL } from './utils/api';
import {
  ComparisonPayload,
  ComparisonResult,
  CreateListPayload,
  ListDetail,
  ListSummary,
  RankingsResponse,
} from './types';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { formatContent } from './utils/format';

const DEFAULT_STATS = {
  comparisons: 0,
  confidence: 0,
  progress: 0,
};

const confidenceFromProgress = (completed: number) => Math.min(100, completed * 5);

type RawListSummary = Omit<ListSummary, 'id'> & { id: string | number };
type RawListDetail = Omit<ListDetail, 'id'> & { id: string | number };
type RawComparisonPayload = Omit<ComparisonPayload, 'item_a' | 'item_b'> & {
  item_a: Omit<ComparisonPayload['item_a'], 'id'> & { id: string | number };
  item_b: Omit<ComparisonPayload['item_b'], 'id'> & { id: string | number };
};
type RawComparisonResult = Omit<ComparisonResult, 'id' | 'winner_id' | 'loser_id'> & {
  id: string | number;
  winner_id: string | number;
  loser_id: string | number;
};

const toId = (value: string | number) => String(value);

const App = () => {
  const [lists, setLists] = useState<ListSummary[]>([]);
  const [currentList, setCurrentList] = useState<ListDetail | null>(null);
  const [comparison, setComparison] = useState<ComparisonPayload | null>(null);
  const [history, setHistory] = useState<ComparisonResult[]>([]);
  const [hasStarted, setHasStarted] = useState(false);
  const [isCompleted, setIsCompleted] = useState(false);
  const [stats, setStats] = useState(DEFAULT_STATS);
  const [isLoadingComparison, setIsLoadingComparison] = useState(false);
  const [rankings, setRankings] = useState<RankingsResponse | null>(null);
  const [rankingsOpen, setRankingsOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [isCreatingList, setIsCreatingList] = useState(false);
  const [errorBanner, setErrorBanner] = useState<string | null>(null);

  const loadLists = useCallback(async () => {
    try {
      const data = await apiFetch<RawListSummary[]>('/lists');
      setLists(data.map((list) => ({ ...list, id: toId(list.id) })));
      setErrorBanner(null);
    } catch (error) {
      console.error('Failed to load lists', error);
      setErrorBanner('We could not load your lists. Refresh or check the API service.');
    }
  }, []);

  const resetSessionState = useCallback(() => {
    setComparison(null);
    setHistory([]);
    setStats(DEFAULT_STATS);
    setHasStarted(false);
    setIsCompleted(false);
  }, []);

  const selectList = useCallback(
    async (id: string) => {
      try {
        const detail = await apiFetch<RawListDetail>(`/lists/${id}`);
        setCurrentList({ ...detail, id: toId(detail.id) });
        setErrorBanner(null);
        resetSessionState();
      } catch (error) {
        console.error('Failed to fetch list', error);
        setErrorBanner('We could not fetch that list. Try again.');
      }
    },
    [resetSessionState],
  );

  const fetchNextComparison = useCallback(async () => {
    if (!currentList) return;
    setIsLoadingComparison(true);

    try {
      const response = await fetch(`${API_BASE_URL}/lists/${currentList.id}/next-comparison`);

      if (response.status === 204 || response.status === 404) {
        setIsCompleted(true);
        setComparison(null);
        setIsLoadingComparison(false);
        return;
      }

      if (!response.ok) {
        throw new Error(await response.text());
      }

      const payload = (await response.json()) as RawComparisonPayload;
      const normalized: ComparisonPayload = {
        ...payload,
        item_a: { ...payload.item_a, id: toId(payload.item_a.id) },
        item_b: { ...payload.item_b, id: toId(payload.item_b.id) },
      };
      setComparison(normalized);
      setErrorBanner(null);

      const completed = normalized.progress?.completed ?? 0;
      const total = normalized.progress?.total ?? 1;
      setStats({
        comparisons: completed,
        confidence: confidenceFromProgress(completed),
        progress: total ? (completed / total) * 100 : 0,
      });
      setIsCompleted(false);
    } catch (error) {
      console.error('Failed to load comparison', error);
      setErrorBanner('Next comparison could not be loaded. Try again.');
    } finally {
      setIsLoadingComparison(false);
    }
  }, [currentList]);

  const startRanking = () => {
    if (!currentList) {
      setCreateOpen(true);
      return;
    }

    setHasStarted(true);
    setIsCompleted(false);
    setHistory([]);
    setComparison(null);
    fetchNextComparison();
  };

  const handleSelectWinner = useCallback(
    async (side: 'left' | 'right') => {
      if (!currentList || !comparison) return;

      const winnerId = side === 'left' ? comparison.item_a.id : comparison.item_b.id;
      const loserId = side === 'left' ? comparison.item_b.id : comparison.item_a.id;

      try {
        const result = await apiFetch<RawComparisonResult>('/comparisons', {
          method: 'POST',
          body: JSON.stringify({
            list_id: currentList.id,
            winner_id: winnerId,
            loser_id: loserId,
          }),
        });

        setHistory((prev) => [
          ...prev,
          { ...result, id: toId(result.id), winner_id: toId(result.winner_id), loser_id: toId(result.loser_id) },
        ]);
        setErrorBanner(null);
        await fetchNextComparison();
      } catch (error) {
        console.error('Failed to submit comparison', error);
        setErrorBanner('We could not record that decision. Please try again.');
      }
    },
    [comparison, currentList, fetchNextComparison],
  );

  const handleSkip = useCallback(() => {
    fetchNextComparison();
  }, [fetchNextComparison]);

  const handleUndo = useCallback(async () => {
    const last = history.at(-1);
    if (!last || !currentList) return;

    setHistory((prev) => prev.slice(0, -1));

    try {
      await apiFetch(`/comparisons/${last.id}`, { method: 'DELETE' });
      setErrorBanner(null);
      await fetchNextComparison();
    } catch (error) {
      console.error('Failed to undo comparison', error);
      setErrorBanner('Undo failed. The last comparison may already be reconciled.');
    }
  }, [history, currentList, fetchNextComparison]);

  const openRankings = useCallback(async () => {
    if (!currentList) return;

    try {
      const data = await apiFetch<RankingsResponse>(`/lists/${currentList.id}/rankings`);
      setRankings(data);
      setRankingsOpen(true);
      setErrorBanner(null);
    } catch (error) {
      console.error('Failed to load rankings', error);
      setErrorBanner('Rankings not available right now.');
    }
  }, [currentList]);

  const exportRankings = useCallback(() => {
    if (!currentList || !rankings) return;

    const rows = [
      ['Rank', 'Item', 'Elo Rating', 'Confidence'],
      ...rankings.rankings.map((entry, index) => [
        `${index + 1}`,
        formatContent(entry.item),
        Math.round(entry.elo_rating).toString(),
        `${Math.round(entry.confidence * 100)}%`,
      ]),
    ];

    const escape = (value: string) => `"${value.replace(/"/g, '""')}"`;
    const csv = rows.map((row) => row.map((col) => escape(col)).join(',')).join('\n');

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });

    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `${currentList.name.replace(/\s+/g, '-').toLowerCase()}-rankings.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
  }, [currentList, rankings]);

  const createList = useCallback(
    async (payload: CreateListPayload) => {
      setIsCreatingList(true);
      try {
        const response = await apiFetch<{ list_id: string | number }>('/lists', {
          method: 'POST',
          body: JSON.stringify(payload),
        });

        await loadLists();
        setCreateOpen(false);
        setIsCreatingList(false);
        setErrorBanner(null);
        await selectList(toId(response.list_id));
      } catch (error) {
        console.error('Failed to create list', error);
        setErrorBanner('Creating the list failed. Double-check your data.');
        setIsCreatingList(false);
      }
    },
    [loadLists, selectList],
  );

  useEffect(() => {
    loadLists();
  }, [loadLists]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const listId = params.get('list');
    if (listId) {
      selectList(listId).catch(() => undefined);
    }
  }, [selectList]);

  useKeyboardShortcuts({
    enabled: hasStarted && !isCompleted && Boolean(comparison),
    onLeft: () => handleSelectWinner('left'),
    onRight: () => handleSelectWinner('right'),
    onSkip: handleSkip,
    onUndo: handleUndo,
  });

  const dropdownDisabled = lists.length === 0;

  const canInteract = hasStarted && !isCompleted && !isLoadingComparison;

  const instructionsDisabled = !currentList;

  return (
    <div className="app-shell">
      <div className="ambient" />

      {errorBanner && (
        <div className="banner banner--error">
          <span>{errorBanner}</span>
          <button type="button" onClick={() => setErrorBanner(null)}>
            Dismiss
          </button>
        </div>
      )}

      <TopBar
        listTrigger={
          <ListPicker lists={lists} currentListId={currentList?.id} onSelect={selectList} disabled={dropdownDisabled} />
        }
        onViewRankings={openRankings}
        onCreateList={() => setCreateOpen(true)}
      />

      <StatusStrip
        comparisons={stats.comparisons}
        confidence={stats.confidence}
        progress={stats.progress}
        activeListName={currentList?.name}
      />

      {!hasStarted ? (
        <InstructionPanel onStart={startRanking} disabled={instructionsDisabled} />
      ) : (
        <ComparisonStage
          comparison={comparison}
          onSelect={handleSelectWinner}
          onSkip={handleSkip}
          onUndo={handleUndo}
          disabled={!canInteract}
          isComplete={isCompleted}
          onShowRankings={openRankings}
          onContinue={() => {
            setIsCompleted(false);
            fetchNextComparison();
          }}
        />
      )}

      <Modal
        open={rankingsOpen}
        onClose={() => setRankingsOpen(false)}
        title="Live rankings"
        subtitle="Ordered by Elo score with confidence multipliers."
        footer={
          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={() => setRankingsOpen(false)}>
              Close
            </button>
            <button type="button" className="btn btn-primary" onClick={exportRankings}>
              Export CSV
            </button>
          </div>
        }
      >
        <RankingsPanel data={rankings} />
      </Modal>

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Create a new comparison list"
        subtitle="Bring clarity to any crowded backlog."
        size="lg"
      >
        <CreateListForm onSubmit={createList} loading={isCreatingList} />
      </Modal>
    </div>
  );
};

export default App;
