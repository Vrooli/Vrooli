import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import ReactMarkdown from "react-markdown";

const overlayVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 },
};

const panelVariants = {
  hidden: { opacity: 0, y: 20, scale: 0.98 },
  visible: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: 20, scale: 0.96 },
};

const pageVariants = {
  enter: (direction) => ({ x: direction > 0 ? 120 : -120, opacity: 0 }),
  center: { x: 0, opacity: 1 },
  exit: (direction) => ({ x: direction < 0 ? 120 : -120, opacity: 0 }),
};

const BookReader = ({ story, onClose }) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [pages, setPages] = useState([]);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [speechRate, setSpeechRate] = useState(0.9); // Slower for children
  const [speechVoice, setSpeechVoice] = useState(null);
  const [availableVoices, setAvailableVoices] = useState([]);

  useEffect(() => {
    const splitPages = story.content.split("## Page");
    const formattedPages = splitPages
      .filter((page) => page.trim())
      .map((page) =>
        page.trim().match(/^\d/)
          ? page.replace(/^\d+/, "").trim()
          : page.trim(),
      );
    setPages(formattedPages);
    setCurrentPage(0);
  }, [story]);

  // Initialize speech synthesis voices
  useEffect(() => {
    const loadVoices = () => {
      const voices = window.speechSynthesis.getVoices();
      setAvailableVoices(voices);
      // Try to find a child-friendly or female voice
      const preferredVoice =
        voices.find(
          (voice) =>
            voice.name.toLowerCase().includes("female") ||
            voice.name.toLowerCase().includes("child") ||
            voice.name.toLowerCase().includes("samantha") ||
            voice.name.toLowerCase().includes("victoria"),
        ) || voices[0];
      setSpeechVoice(preferredVoice);
    };

    loadVoices();
    // Some browsers load voices asynchronously
    window.speechSynthesis.onvoiceschanged = loadVoices;

    return () => {
      window.speechSynthesis.cancel();
    };
  }, []);

  const speakText = (text) => {
    if ("speechSynthesis" in window) {
      // Cancel any ongoing speech
      window.speechSynthesis.cancel();

      // Remove markdown formatting for cleaner speech
      const cleanText = text
        .replace(/\*\*/g, "") // Remove bold markers
        .replace(/##/g, "") // Remove headers
        .replace(/\n\n/g, ". ") // Replace double newlines with periods
        .replace(/\n/g, " "); // Replace single newlines with spaces

      const utterance = new SpeechSynthesisUtterance(cleanText);
      utterance.rate = speechRate;
      utterance.pitch = 1.1; // Slightly higher pitch for friendlier sound
      if (speechVoice) {
        utterance.voice = speechVoice;
      }

      utterance.onstart = () => setIsSpeaking(true);
      utterance.onend = () => setIsSpeaking(false);
      utterance.onerror = () => setIsSpeaking(false);

      window.speechSynthesis.speak(utterance);
    }
  };

  const toggleSpeech = () => {
    if (isSpeaking) {
      window.speechSynthesis.cancel();
      setIsSpeaking(false);
    } else {
      speakText(pages[currentPage]);
    }
  };

  const nextPage = () => {
    if (currentPage < pages.length - 1) {
      window.speechSynthesis.cancel();
      setIsSpeaking(false);
      setCurrentPage((prev) => prev + 1);
    }
  };

  const prevPage = () => {
    if (currentPage > 0) {
      window.speechSynthesis.cancel();
      setIsSpeaking(false);
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
      <motion.div
        className="reader-panel"
        role="dialog"
        aria-modal="true"
        variants={panelVariants}
      >
        <button
          className="icon-button close"
          onClick={onClose}
          aria-label="Close story"
        >
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
            <span>
              Page {pages.length ? currentPage + 1 : 0} of {pages.length || 0}
            </span>
            <div className="progress-track">
              <div
                className="progress-thumb"
                style={{ width: `${progress}%` }}
              />
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
                <ReactMarkdown>{pages[currentPage] || ""}</ReactMarkdown>
              </motion.div>
            </AnimatePresence>
          </div>

          <aside className="reader-aside">
            <div className="aside-card">
              <h3>Bedtime tip</h3>
              <p>
                Read slowly and pause after each page to ask your listener what
                they imagine. It keeps bedtime calm and collaborative.
              </p>
            </div>
            <div className="aside-card">
              <h3>Story stats</h3>
              <ul>
                <li>
                  <strong>Reads:</strong> {story.times_read || 0}
                </li>
                {story.last_read && (
                  <li>
                    <strong>Last shared:</strong>{" "}
                    {new Date(story.last_read).toLocaleDateString([], {
                      month: "short",
                      day: "numeric",
                    })}
                  </li>
                )}
              </ul>
            </div>
          </aside>
        </section>

        <footer className="reader-controls">
          <button
            className="ghost-action"
            onClick={prevPage}
            disabled={currentPage === 0}
          >
            ← Previous
          </button>

          <button
            className={`speech-button ${isSpeaking ? "speaking" : ""}`}
            onClick={toggleSpeech}
            title={isSpeaking ? "Stop reading" : "Read aloud"}
            aria-label={
              isSpeaking ? "Stop reading aloud" : "Start reading aloud"
            }
          >
            {isSpeaking ? "🔊 Stop" : "🔈 Read Aloud"}
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
