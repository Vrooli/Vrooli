import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export function GenerateQuiz() {
  const navigate = useNavigate();
  const [content, setContent] = useState('');
  const [questionCount, setQuestionCount] = useState(10);
  const [difficulty, setDifficulty] = useState('mixed');
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');

  const handleGenerate = async () => {
    if (!content.trim()) {
      setError('Please provide some content to generate questions from');
      return;
    }

    setGenerating(true);
    setError('');

    try {
      const response = await fetch('/api/v1/quiz/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content,
          question_count: questionCount,
          difficulty,
          question_types: ['mcq', 'true_false', 'short_answer']
        })
      });

      if (!response.ok) throw new Error('Failed to generate quiz');

      const data = await response.json();
      navigate(`/quiz/${data.quiz_id}`);
    } catch (err) {
      setError('Failed to generate quiz. Please try again.');
      setGenerating(false);
    }
  };

  return (
    <div className="generate-quiz">
      <h1>Generate Quiz with AI</h1>

      {error && <div className="alert alert-error">{error}</div>}

      <div className="form-group">
        <label>Content to Generate From</label>
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Paste your text content here..."
          className="input textarea"
          rows={10}
        />
      </div>

      <div className="form-row">
        <div className="form-group">
          <label>Number of Questions</label>
          <input
            type="number"
            min="5"
            max="50"
            value={questionCount}
            onChange={(e) => setQuestionCount(parseInt(e.target.value))}
            className="input"
          />
        </div>

        <div className="form-group">
          <label>Difficulty</label>
          <select
            value={difficulty}
            onChange={(e) => setDifficulty(e.target.value)}
            className="input"
          >
            <option value="easy">Easy</option>
            <option value="medium">Medium</option>
            <option value="hard">Hard</option>
            <option value="mixed">Mixed</option>
          </select>
        </div>
      </div>

      <button
        onClick={handleGenerate}
        disabled={generating || !content.trim()}
        className="button button-primary button-large"
      >
        {generating ? 'Generating...' : '🤖 Generate Quiz'}
      </button>
    </div>
  );
}