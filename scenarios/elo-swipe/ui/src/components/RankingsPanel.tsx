import { formatContent } from '../utils/format';
import type { RankingsResponse } from '../types';

interface RankingsPanelProps {
  data: RankingsResponse | null;
}

export const RankingsPanel = ({ data }: RankingsPanelProps) => {
  if (!data || data.rankings.length === 0) {
    return <p className="rankings-empty">No rankings yet. Generate comparisons to populate the leaderboard.</p>;
  }

  return (
    <ul className="rankings-list">
      {data.rankings.map((entry, index) => (
        <li key={`${entry.elo_rating}-${index}`} className="rankings-item">
          <div className="rankings-item__position">{index + 1}</div>
          <div className="rankings-item__content">
            <div className="rankings-item__title">{formatContent(entry.item)}</div>
            <div className="rankings-item__meta">
              <span>
                Elo score <strong>{Math.round(entry.elo_rating)}</strong>
              </span>
              <span>
                Confidence <strong>{Math.round(entry.confidence * 100)}%</strong>
              </span>
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
};
