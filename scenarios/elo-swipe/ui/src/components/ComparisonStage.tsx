import { AnimatePresence, motion } from 'framer-motion';
import type { ComparisonPayload } from '../types';
import { formatContent } from '../utils/format';
import { ArrowPathIcon, ForwardIcon } from '@heroicons/react/24/outline';

interface ComparisonStageProps {
  comparison: ComparisonPayload | null;
  onSelect: (side: 'left' | 'right') => void;
  onSkip: () => void;
  onUndo: () => void;
  disabled?: boolean;
  isComplete?: boolean;
  onShowRankings: () => void;
  onContinue: () => void;
}

const swipeThreshold = 120;

export const ComparisonStage = ({
  comparison,
  onSelect,
  onSkip,
  onUndo,
  disabled,
  isComplete,
  onShowRankings,
  onContinue,
}: ComparisonStageProps) => {
  if (isComplete) {
    return (
      <section className="comparison-stage panel comparison-stage--complete">
        <div className="completion-card">
          <h2>Ranking run complete</h2>
          <p>
            You have generated enough pairwise signal to lock in a confident ordering. Review the standings
            or keep exploring new match-ups to further tighten confidence.
          </p>
          <div className="completion-actions">
            <button type="button" className="btn btn-primary" onClick={onShowRankings}>
              Review rankings
            </button>
            <button type="button" className="btn btn-ghost" onClick={onContinue}>
              Keep comparing
            </button>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="comparison-stage panel">
      <AnimatePresence mode="wait">
        {comparison ? (
          <div key={comparison.item_a.id + comparison.item_b.id} className="comparison-cards">
            <motion.button
              type="button"
              key={comparison.item_a.id}
              className="comparison-card comparison-card--left"
              onClick={() => onSelect('left')}
              drag="x"
              dragConstraints={{ left: 0, right: 0 }}
              dragElastic={0.2}
              dragSnapToOrigin
              onDragEnd={(_, info) => {
                if (info.offset.x < -swipeThreshold) onSelect('left');
              }}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -30 }}
              whileTap={{ scale: 0.97 }}
              disabled={disabled}
            >
              <span className="comparison-card__label left">Option A</span>
              <span className="comparison-card__content">{formatContent(comparison.item_a.content)}</span>
            </motion.button>

            <motion.div className="comparison-divider" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
              <span>vs</span>
            </motion.div>

            <motion.button
              type="button"
              key={comparison.item_b.id}
              className="comparison-card comparison-card--right"
              onClick={() => onSelect('right')}
              drag="x"
              dragConstraints={{ left: 0, right: 0 }}
              dragElastic={0.2}
              dragSnapToOrigin
              onDragEnd={(_, info) => {
                if (info.offset.x > swipeThreshold) onSelect('right');
              }}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -30 }}
              whileTap={{ scale: 0.97 }}
              disabled={disabled}
            >
              <span className="comparison-card__label right">Option B</span>
              <span className="comparison-card__content">{formatContent(comparison.item_b.content)}</span>
            </motion.button>
          </div>
        ) : (
          <motion.div key="loading-state" className="comparison-empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
            <ForwardIcon />
            <p>We are fetching the next matchup for you...</p>
          </motion.div>
        )}
      </AnimatePresence>

      <footer className="comparison-controls">
        <button type="button" className="pill pill--accent" onClick={onSkip} disabled={disabled}>
          <ForwardIcon />
          <span>Skip</span>
          <kbd>Space</kbd>
        </button>
        <button type="button" className="pill" onClick={onUndo} disabled={disabled}>
          <ArrowPathIcon />
          <span>Undo</span>
          <kbd>Ctrl+Z</kbd>
        </button>
      </footer>
    </section>
  );
};
