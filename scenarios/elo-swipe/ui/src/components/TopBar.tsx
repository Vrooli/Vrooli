import { BoltIcon, PlusIcon, TrophyIcon } from '@heroicons/react/24/outline';
import type { ReactNode } from 'react';

interface TopBarProps {
  listTrigger: ReactNode;
  onViewRankings: () => void;
  onCreateList: () => void;
}

export const TopBar = ({ listTrigger, onViewRankings, onCreateList }: TopBarProps) => {
  return (
    <header className="topbar panel">
      <div className="topbar__brand">
        <div className="brand-mark">
          <BoltIcon />
        </div>
        <div>
          <h1>Elo Swipe</h1>
          <p>Signal-first ranking intelligence.</p>
        </div>
      </div>
      <div className="topbar__actions">
        {listTrigger}
        <button type="button" className="btn btn-ghost" onClick={onViewRankings}>
          <TrophyIcon />
          Live Rankings
        </button>
        <button type="button" className="btn btn-primary" onClick={onCreateList}>
          <PlusIcon />
          New List
        </button>
      </div>
    </header>
  );
};
