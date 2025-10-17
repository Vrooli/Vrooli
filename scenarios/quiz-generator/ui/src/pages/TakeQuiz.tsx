import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';

interface Quiz {
  id: string;
  title: string;
  description?: string;
  questions: Question[];
  time_limit?: number;
  passing_score: number;
}

interface Question {
  id: string;
  type: string;
  question: string;
  options?: string[];
  difficulty: string;
  points: number;
  order_index: number;
}

interface QuestionResponse {
  question_id: string;
  answer: string;
  isCorrect?: boolean;
  explanation?: string;
}

export function TakeQuiz() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [quiz, setQuiz] = useState<Quiz | null>(null);
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [responses, setResponses] = useState<QuestionResponse[]>([]);
  const [currentAnswer, setCurrentAnswer] = useState('');
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedbackMessage, setFeedbackMessage] = useState('');
  const [practiceMode, setPracticeMode] = useState(true); // Real-time feedback mode
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [startTime, setStartTime] = useState<Date | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchQuiz();
  }, [id]);

  const fetchQuiz = async () => {
    try {
      const response = await fetch(`/api/v1/quiz/${id}`);
      if (!response.ok) throw new Error('Quiz not found');
      const data = await response.json();
      setQuiz(data);
      setStartTime(new Date());
    } catch (err) {
      setError('Failed to load quiz');
    } finally {
      setLoading(false);
    }
  };

  const handleAnswerSubmit = async () => {
    if (!quiz || !currentAnswer.trim()) return;

    const currentQuestion = quiz.questions[currentQuestionIndex];

    // In practice mode, check answer immediately
    if (practiceMode) {
      // For demo purposes, we'll simulate checking
      // In production, this would call an API endpoint
      const isCorrect = simulateAnswerCheck(currentQuestion, currentAnswer);

      setShowFeedback(true);
      setFeedbackMessage(
        isCorrect
          ? '✅ Correct! Well done!'
          : '❌ Incorrect. The right answer will be shown after completing the quiz.'
      );

      // Store the response
      const response: QuestionResponse = {
        question_id: currentQuestion.id,
        answer: currentAnswer,
        isCorrect,
        explanation: isCorrect ? 'Great job!' : 'Review this topic'
      };

      setResponses([...responses, response]);

      // Auto-advance after 2 seconds
      setTimeout(() => {
        if (currentQuestionIndex < quiz.questions.length - 1) {
          setCurrentQuestionIndex(currentQuestionIndex + 1);
          setCurrentAnswer('');
          setShowFeedback(false);
          setFeedbackMessage('');
        } else {
          // Quiz complete - submit results
          submitQuiz([...responses, response]);
        }
      }, 2000);
    } else {
      // Test mode - just store answer and move on
      const response: QuestionResponse = {
        question_id: currentQuestion.id,
        answer: currentAnswer
      };

      const newResponses = [...responses, response];
      setResponses(newResponses);

      if (currentQuestionIndex < quiz.questions.length - 1) {
        setCurrentQuestionIndex(currentQuestionIndex + 1);
        setCurrentAnswer('');
      } else {
        submitQuiz(newResponses);
      }
    }
  };

  const simulateAnswerCheck = (question: Question, answer: string): boolean => {
    // This is a simplified check for demo
    // In production, the API would handle this
    if (question.type === 'true_false') {
      return answer.toLowerCase() === 'true' || answer.toLowerCase() === 'false';
    }
    // For MCQ, check if answer matches one of the options
    if (question.type === 'mcq' && question.options) {
      return question.options.includes(answer);
    }
    // For other types, accept any non-empty answer
    return answer.trim().length > 0;
  };

  const submitQuiz = async (allResponses: QuestionResponse[]) => {
    if (!startTime) return;

    setSubmitting(true);
    const timeTaken = Math.floor((new Date().getTime() - startTime.getTime()) / 1000);

    try {
      const response = await fetch(`/api/v1/quiz/${id}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          responses: allResponses.map(r => ({
            question_id: r.question_id,
            answer: r.answer
          })),
          time_taken: timeTaken
        })
      });

      if (!response.ok) throw new Error('Failed to submit quiz');

      const result = await response.json();

      // Store result and navigate to results page
      sessionStorage.setItem(`quiz-result-${id}`, JSON.stringify({
        ...result,
        responses: allResponses,
        quiz: quiz,
        time_taken: timeTaken
      }));

      navigate(`/results/${id}`);
    } catch (err) {
      setError('Failed to submit quiz');
      setSubmitting(false);
    }
  };

  if (loading) return <div className="loading">Loading quiz...</div>;
  if (error) return <div className="error">{error}</div>;
  if (!quiz) return <div className="error">Quiz not found</div>;

  const currentQuestion = quiz.questions[currentQuestionIndex];
  const progress = ((currentQuestionIndex + 1) / quiz.questions.length) * 100;

  return (
    <div className="take-quiz">
      <div className="quiz-header">
        <h1>{quiz.title}</h1>
        {quiz.description && <p>{quiz.description}</p>}

        <div className="quiz-controls">
          <label className="practice-toggle">
            <input
              type="checkbox"
              checked={practiceMode}
              onChange={(e) => setPracticeMode(e.target.checked)}
            />
            <span>Practice Mode (instant feedback)</span>
          </label>
        </div>

        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${progress}%` }}
          />
        </div>
        <div className="progress-text">
          Question {currentQuestionIndex + 1} of {quiz.questions.length}
        </div>
      </div>

      <div className="question-container">
        <div className="question-header">
          <span className="question-type">{currentQuestion.type.replace('_', ' ').toUpperCase()}</span>
          <span className="question-difficulty difficulty-{currentQuestion.difficulty}">
            {currentQuestion.difficulty}
          </span>
          <span className="question-points">{currentQuestion.points} points</span>
        </div>

        <h2 className="question-text">{currentQuestion.question}</h2>

        <div className="answer-section">
          {currentQuestion.type === 'mcq' && currentQuestion.options && (
            <div className="options-list">
              {currentQuestion.options.map((option, index) => (
                <label key={index} className="option-label">
                  <input
                    type="radio"
                    name="answer"
                    value={option}
                    checked={currentAnswer === option}
                    onChange={(e) => setCurrentAnswer(e.target.value)}
                    disabled={showFeedback}
                  />
                  <span className="option-text">{option}</span>
                </label>
              ))}
            </div>
          )}

          {currentQuestion.type === 'true_false' && (
            <div className="options-list">
              <label className="option-label">
                <input
                  type="radio"
                  name="answer"
                  value="true"
                  checked={currentAnswer === 'true'}
                  onChange={(e) => setCurrentAnswer(e.target.value)}
                  disabled={showFeedback}
                />
                <span className="option-text">True</span>
              </label>
              <label className="option-label">
                <input
                  type="radio"
                  name="answer"
                  value="false"
                  checked={currentAnswer === 'false'}
                  onChange={(e) => setCurrentAnswer(e.target.value)}
                  disabled={showFeedback}
                />
                <span className="option-text">False</span>
              </label>
            </div>
          )}

          {(currentQuestion.type === 'short_answer' || currentQuestion.type === 'fill_blank') && (
            <div className="text-answer">
              <textarea
                value={currentAnswer}
                onChange={(e) => setCurrentAnswer(e.target.value)}
                placeholder="Enter your answer here..."
                className="answer-input"
                rows={4}
                disabled={showFeedback}
              />
            </div>
          )}
        </div>

        {showFeedback && (
          <div className={`feedback ${feedbackMessage.includes('✅') ? 'feedback-correct' : 'feedback-incorrect'}`}>
            {feedbackMessage}
          </div>
        )}

        <div className="question-actions">
          <button
            onClick={handleAnswerSubmit}
            disabled={!currentAnswer.trim() || showFeedback || submitting}
            className="button button-primary"
          >
            {currentQuestionIndex === quiz.questions.length - 1 ? 'Finish Quiz' : 'Next Question'}
          </button>
        </div>
      </div>

      <div className="quiz-stats">
        <div className="stat">
          <span className="stat-label">Answered:</span>
          <span className="stat-value">{responses.length} / {quiz.questions.length}</span>
        </div>
        {practiceMode && (
          <div className="stat">
            <span className="stat-label">Correct:</span>
            <span className="stat-value">
              {responses.filter(r => r.isCorrect).length}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}