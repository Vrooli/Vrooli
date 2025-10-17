import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';

export function QuizResults() {
  const { id } = useParams<{ id: string }>();
  const [result, setResult] = useState<any>(null);

  useEffect(() => {
    const storedResult = sessionStorage.getItem(`quiz-result-${id}`);
    if (storedResult) {
      setResult(JSON.parse(storedResult));
    }
  }, [id]);

  if (!result) {
    return <div className="error">Results not found</div>;
  }

  return (
    <div className="quiz-results">
      <h1>Quiz Complete!</h1>
      
      <div className="result-summary">
        <div className="score-circle">
          <div className="score-value">{result.percentage?.toFixed(0) || 0}%</div>
          <div className="score-label">{result.passed ? 'PASSED' : 'FAILED'}</div>
        </div>
      </div>

      <div className="results-stats">
        <div className="stat">
          <span>Score:</span>
          <strong>{result.score} points</strong>
        </div>
        <div className="stat">
          <span>Time:</span>
          <strong>{Math.floor(result.time_taken / 60)}m {result.time_taken % 60}s</strong>
        </div>
        <div className="stat">
          <span>Correct:</span>
          <strong>{result.responses?.filter((r: any) => r.isCorrect).length || 0} / {result.responses?.length || 0}</strong>
        </div>
      </div>

      <div className="action-buttons">
        <Link to={`/quiz/${id}`} className="button button-primary">
          Retake Quiz
        </Link>
        <Link to="/" className="button button-secondary">
          Back to Dashboard
        </Link>
      </div>
    </div>
  );
}