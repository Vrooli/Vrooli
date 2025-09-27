import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import ReactMarkdown from 'react-markdown';

const overlayVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 }
};

const panelVariants = {
  hidden: { opacity: 0, y: 20, scale: 0.98 },
  visible: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: 20, scale: 0.96 }
};

const pageVariants = {
  enter: (direction) => ({ x: direction > 0 ? 120 : -120, opacity: 0 }),
  center: { x: 0, opacity: 1 },
  exit: (direction) => ({ x: direction < 0 ? 120 : -120, opacity: 0 })
};

const BookReader = ({ story, onClose }) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [pages, setPages] = useState([]);

  useEffect(() => {
    const splitPages = story.content.split('## Page');
    const formattedPages = splitPages
      .filter((page) => page.trim())
      .map((page) => (page.trim().match(/^\d/) ? page.replace(/^\d+/, '').trim() : page.trim()));
    setPages(formattedPages);
    setCurrentPage(0);
  }, [story]);

  const nextPage = () => {
    if (currentPage < pages.length - 1) {
      setCurrentPage((prev) => prev + 1);
    }
  };

  const prevPage = () => {
    if (currentPage > 0) {
      setCurrentPage((prev) => prev - 1);
    }
  };

  const progress = pages.length ? ((currentPage + 1) / pages.length) * 100 : 0;

  return (
    <motion.div
      className="modal-overlay reader"
      initial="hidden"
      animate="visible"
      exit="exit"
      variants={overlayVariants}
    >
      <motion.div className="reader-panel" role="dialog" aria-modal="true" variants={panelVariants}>
        <button className="icon-button close" onClick={onClose} aria-label="Close story">
          ✕
        </button>

        <header className="reader-header">
          <div>
            <h2>{story.title}</h2>
            <div className="reader-meta">
              <span>📖 {story.reading_time_minutes || 5} min read</span>
              {story.theme && <span>🎨 {story.theme}</span>}
              <span>👶 Ages {story.age_group}</span>
            </div>
          </div>
          <div className="reader-progress">
            <span>Page {pages.length ? currentPage + 1 : 0} of {pages.length || 0}</span>
            <div className="progress-track">
              <div className="progress-thumb" style={{ width: `${progress}%` }} />
            </div>
          </div>
        </header>

        <section className="reader-body">
          <div className="reader-book">
            <AnimatePresence mode="wait" custom={currentPage}>
              <motion.div
                key={currentPage}
                custom={currentPage}
                variants={pageVariants}
                initial="enter"
                animate="center"
                exit="exit"
                transition={{ duration: 0.4 }}
                className="reader-page"
              >
                <ReactMarkdown>{pages[currentPage] || ''}</ReactMarkdown>
              </motion.div>
            </AnimatePresence>
          </div>

          <aside className="reader-aside">
            <div className="aside-card">
              <h3>Bedtime tip</h3>
              <p>
                Read slowly and pause after each page to ask your listener what they imagine. It keeps
                bedtime calm and collaborative.
              </p>
            </div>
            <div className="aside-card">
              <h3>Story stats</h3>
              <ul>
                <li><strong>Reads:</strong> {story.times_read || 0}</li>
                {story.last_read && (
                  <li>
                    <strong>Last shared:</strong>{' '}
                    {new Date(story.last_read).toLocaleDateString([], { month: 'short', day: 'numeric' })}
                  </li>
                )}
              </ul>
            </div>
          </aside>
        </section>

        <footer className="reader-controls">
          <button className="ghost-action" onClick={prevPage} disabled={currentPage === 0}>
            ← Previous
          </button>
          {currentPage === pages.length - 1 ? (
            <button className="primary-action" onClick={onClose}>
              Back to library
            </button>
          ) : (
            <button className="primary-action" onClick={nextPage}>
              Next page →
            </button>
          )}
        </footer>
      </motion.div>
    </motion.div>
  );
};

export default BookReader;
