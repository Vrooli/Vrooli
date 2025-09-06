import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import ReactMarkdown from 'react-markdown';

const BookReader = ({ story, onClose, theme }) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [pages, setPages] = useState([]);

  useEffect(() => {
    // Split content into pages
    const pageBreaks = story.content.split('## Page');
    const formattedPages = pageBreaks
      .filter(page => page.trim())
      .map(page => {
        // Check if this is the first page (no page number prefix)
        if (!page.trim().match(/^\d/)) {
          return page.trim();
        }
        // Remove the page number from the beginning
        return page.replace(/^\d+/, '').trim();
      });
    
    setPages(formattedPages);
    setCurrentPage(0);
  }, [story]);

  const nextPage = () => {
    if (currentPage < pages.length - 1) {
      setCurrentPage(currentPage + 1);
    }
  };

  const prevPage = () => {
    if (currentPage > 0) {
      setCurrentPage(currentPage - 1);
    }
  };

  const pageVariants = {
    enter: (direction) => ({
      x: direction > 0 ? 1000 : -1000,
      opacity: 0
    }),
    center: {
      zIndex: 1,
      x: 0,
      opacity: 1
    },
    exit: (direction) => ({
      zIndex: 0,
      x: direction < 0 ? 1000 : -1000,
      opacity: 0
    })
  };

  return (
    <motion.div
      className="book-reader"
      initial={{ scale: 0, rotateY: -180 }}
      animate={{ scale: 1, rotateY: 0 }}
      exit={{ scale: 0, rotateY: 180 }}
      transition={{ duration: 0.5 }}
    >
      <button className="close-book-btn" onClick={onClose}>
        ✕
      </button>

      <div className="book-page book-page-left">
        <div className="page-content">
          <h1>{story.title}</h1>
          
          <div style={{ 
            marginBottom: '20px', 
            fontSize: '0.9rem',
            color: '#8b6f47',
            textAlign: 'center'
          }}>
            <span>📖 {story.reading_time_minutes} min read</span>
            {story.theme && <span> • 🎨 {story.theme}</span>}
            <span> • 👶 Ages {story.age_group}</span>
          </div>

          <AnimatePresence mode="wait" custom={currentPage}>
            <motion.div
              key={currentPage}
              custom={currentPage}
              variants={pageVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={{
                x: { type: "spring", stiffness: 300, damping: 30 },
                opacity: { duration: 0.2 }
              }}
            >
              <ReactMarkdown>{pages[currentPage] || ''}</ReactMarkdown>
            </motion.div>
          </AnimatePresence>
        </div>

        <div className="page-number">
          Page {currentPage + 1} of {pages.length}
        </div>
      </div>

      <div className="book-page book-page-right">
        <div className="page-content" style={{ 
          display: 'flex', 
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          {currentPage === pages.length - 1 ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.5 }}
              style={{ textAlign: 'center' }}
            >
              <h2 style={{ 
                fontFamily: 'Grandstander, cursive',
                fontSize: '2.5rem',
                color: '#8b6f47',
                marginBottom: '20px'
              }}>
                The End
              </h2>
              <div style={{ fontSize: '3rem', marginBottom: '20px' }}>
                🌙✨🌟
              </div>
              <p style={{ 
                fontSize: '1.2rem',
                color: '#6d5638',
                fontStyle: 'italic'
              }}>
                Sweet dreams!
              </p>
              <button
                onClick={onClose}
                style={{
                  marginTop: '30px',
                  padding: '10px 20px',
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  color: 'white',
                  border: 'none',
                  borderRadius: '25px',
                  fontSize: '1rem',
                  cursor: 'pointer'
                }}
              >
                Back to Bookshelf
              </button>
            </motion.div>
          ) : (
            <div style={{ 
              width: '100%',
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '5rem',
              opacity: 0.1
            }}>
              {currentPage % 2 === 0 ? '🌙' : '⭐'}
            </div>
          )}
        </div>

        {/* Page navigation */}
        <div style={{
          position: 'absolute',
          bottom: '20px',
          left: '50%',
          transform: 'translateX(-50%)',
          display: 'flex',
          gap: '20px'
        }}>
          <button
            onClick={prevPage}
            disabled={currentPage === 0}
            style={{
              padding: '10px 20px',
              background: currentPage === 0 ? '#e0e0e0' : '#8b6f47',
              color: currentPage === 0 ? '#999' : 'white',
              border: 'none',
              borderRadius: '20px',
              cursor: currentPage === 0 ? 'not-allowed' : 'pointer',
              fontSize: '1rem',
              transition: 'all 0.3s ease'
            }}
          >
            ← Previous
          </button>
          
          <button
            onClick={nextPage}
            disabled={currentPage === pages.length - 1}
            style={{
              padding: '10px 20px',
              background: currentPage === pages.length - 1 ? '#e0e0e0' : '#8b6f47',
              color: currentPage === pages.length - 1 ? '#999' : 'white',
              border: 'none',
              borderRadius: '20px',
              cursor: currentPage === pages.length - 1 ? 'not-allowed' : 'pointer',
              fontSize: '1rem',
              transition: 'all 0.3s ease'
            }}
          >
            Next →
          </button>
        </div>
      </div>
    </motion.div>
  );
};

export default BookReader;