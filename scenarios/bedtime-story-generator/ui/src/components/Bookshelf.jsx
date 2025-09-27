import React, { useMemo } from 'react';
import { motion } from 'framer-motion';
import { getReadingTimeMessage } from '../utils/timeUtils';

const fadeIn = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.4 }
};

const getStoryPreview = (content = '') => {
  const clean = content.replace(/[#*_`>-]/g, ' ').replace(/\s+/g, ' ').trim();
  return clean.length > 160 ? `${clean.slice(0, 160)}…` : clean;
};

const Bookshelf = ({
  stories,
  onSelectStory,
  onGenerateNew,
  onToggleFavorite,
  onDeleteStory,
  onCloseLibrary,
  loading
}) => {
  const orderedStories = useMemo(() => {
    if (!stories?.length) return [];
    return [...stories].sort((a, b) => {
      if (a.is_favorite === b.is_favorite) {
        const aTime = a.updated_at || a.created_at;
        const bTime = b.updated_at || b.created_at;
        if (!aTime || !bTime) return 0;
        return new Date(bTime) - new Date(aTime);
      }
      return a.is_favorite ? -1 : 1;
    });
  }, [stories]);

  if (loading) {
    return (
      <motion.section className="library-panel" {...fadeIn}>
        <div className="panel-header">
          <div>
            <h2>Bedtime Library</h2>
            <p>Gathering your saved adventures…</p>
          </div>
        </div>
        <div className="panel-loading">
          <div className="spinner" />
        </div>
      </motion.section>
    );
  }

  return (
    <motion.section className="library-panel" {...fadeIn}>
      <div className="panel-header">
        <div>
          <h2>Bedtime Library</h2>
          <p>Browse calming adventures or spin up something new.</p>
        </div>
        <div className="panel-header-actions">
          <button className="ghost-action" onClick={onCloseLibrary}>
            Hide
          </button>
          <button className="secondary-action" onClick={onGenerateNew}>
            + New Story
          </button>
        </div>
      </div>

      {orderedStories.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📖</div>
          <h3>Your shelf is empty</h3>
          <p>Start the bedtime ritual by crafting a personalised story.</p>
          <button className="primary-action" onClick={onGenerateNew}>
            Craft the first story
          </button>
        </div>
      ) : (
        <div className="story-grid">
          {orderedStories.map((story, index) => {
            const preview = getStoryPreview(story.content);
            const lastRead = story.last_read ? new Date(story.last_read) : null;
            return (
              <motion.article
                key={story.id}
                className={`story-card ${story.is_favorite ? 'favorite' : ''}`}
                initial={{ opacity: 0, y: 30, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ delay: index * 0.04 }}
              >
                <div className="story-card-top">
                  <span className="story-theme">{story.theme || 'Original'}</span>
                  <button
                    className="icon-button"
                    aria-label={story.is_favorite ? 'Remove from favourites' : 'Mark as favourite'}
                    onClick={(e) => {
                      e.stopPropagation();
                      onToggleFavorite(story.id);
                    }}
                  >
                    {story.is_favorite ? '💜' : '🤍'}
                  </button>
                </div>

                <button
                  className="story-title"
                  onClick={() => onSelectStory(story)}
                >
                  {story.title}
                </button>

                <p className="story-preview">{preview}</p>

                <div className="story-meta">
                  <span>{story.age_group || 'Ages 3-10'}</span>
                  <span>{story.reading_time_minutes || 5} min</span>
                  {lastRead && (
                    <span>
                      Read {lastRead.toLocaleDateString([], { month: 'short', day: 'numeric' })}
                    </span>
                  )}
                </div>

                <div className="story-guidance">
                  {getReadingTimeMessage(story.reading_time_minutes || 5)}
                </div>

                <div className="story-actions">
                  <button
                    className="primary-action"
                    onClick={() => onSelectStory(story)}
                  >
                    Read Together
                  </button>
                  <button
                    className="ghost-action"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDeleteStory(story.id);
                    }}
                  >
                    Remove
                  </button>
                </div>
              </motion.article>
            );
          })}
        </div>
      )}
    </motion.section>
  );
};

export default Bookshelf;
