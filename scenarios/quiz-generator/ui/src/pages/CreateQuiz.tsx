import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

interface Question {
  type: string;
  question: string;
  options?: string[];
  correct_answer: string;
  explanation?: string;
  difficulty: string;
  points: number;
}

export function CreateQuiz() {
  const navigate = useNavigate();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [passingScore, setPassingScore] = useState(70);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [currentQuestion, setCurrentQuestion] = useState<Question>({
    type: 'mcq',
    question: '',
    options: ['', '', '', ''],
    correct_answer: '',
    explanation: '',
    difficulty: 'medium',
    points: 2
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const addQuestion = () => {
    if (!currentQuestion.question.trim()) {
      setError('Question text is required');
      return;
    }
    if (!currentQuestion.correct_answer.trim()) {
      setError('Correct answer is required');
      return;
    }

    setQuestions([...questions, currentQuestion]);
    setCurrentQuestion({
      type: 'mcq',
      question: '',
      options: ['', '', '', ''],
      correct_answer: '',
      explanation: '',
      difficulty: 'medium',
      points: 2
    });
    setError('');
  };

  const removeQuestion = (index: number) => {
    setQuestions(questions.filter((_, i) => i !== index));
  };

  const updateOption = (index: number, value: string) => {
    const newOptions = [...(currentQuestion.options || [])];
    newOptions[index] = value;
    setCurrentQuestion({ ...currentQuestion, options: newOptions });
  };

  const saveQuiz = async () => {
    if (!title.trim()) {
      setError('Quiz title is required');
      return;
    }
    if (questions.length === 0) {
      setError('At least one question is required');
      return;
    }

    setSaving(true);
    setError('');

    try {
      const response = await fetch('/api/v1/quiz', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title,
          description,
          questions,
          passing_score: passingScore
        })
      });

      if (!response.ok) throw new Error('Failed to create quiz');

      const quiz = await response.json();
      navigate(`/quiz/${quiz.id}`);
    } catch (err) {
      setError('Failed to save quiz. Please try again.');
      setSaving(false);
    }
  };

  return (
    <div className="create-quiz">
      <div className="page-header">
        <h1>Create Quiz Manually</h1>
      </div>

      {error && (
        <div className="alert alert-error">
          {error}
        </div>
      )}

      <div className="quiz-form">
        <div className="form-section">
          <h2>Quiz Details</h2>
          <div className="form-group">
            <label htmlFor="title">Title</label>
            <input
              id="title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Enter quiz title"
              className="input"
            />
          </div>
          <div className="form-group">
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Enter quiz description (optional)"
              className="input textarea"
              rows={3}
            />
          </div>
          <div className="form-group">
            <label htmlFor="passing-score">Passing Score (%)</label>
            <input
              id="passing-score"
              type="number"
              min="0"
              max="100"
              value={passingScore}
              onChange={(e) => setPassingScore(parseInt(e.target.value))}
              className="input"
            />
          </div>
        </div>

        <div className="form-section">
          <h2>Questions ({questions.length})</h2>

          {questions.map((q, index) => (
            <div key={index} className="question-preview">
              <div className="question-header">
                <span>Q{index + 1}: {q.question}</span>
                <button
                  onClick={() => removeQuestion(index)}
                  className="button button-small button-danger"
                >
                  Remove
                </button>
              </div>
              <div className="question-meta">
                {q.type} • {q.difficulty} • {q.points} points
              </div>
            </div>
          ))}
        </div>

        <div className="form-section">
          <h2>Add Question</h2>

          <div className="form-group">
            <label htmlFor="question-type">Question Type</label>
            <select
              id="question-type"
              value={currentQuestion.type}
              onChange={(e) => setCurrentQuestion({ ...currentQuestion, type: e.target.value })}
              className="input"
            >
              <option value="mcq">Multiple Choice</option>
              <option value="true_false">True/False</option>
              <option value="short_answer">Short Answer</option>
              <option value="fill_blank">Fill in the Blank</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="question-text">Question</label>
            <textarea
              id="question-text"
              value={currentQuestion.question}
              onChange={(e) => setCurrentQuestion({ ...currentQuestion, question: e.target.value })}
              placeholder="Enter your question"
              className="input textarea"
              rows={2}
            />
          </div>

          {currentQuestion.type === 'mcq' && (
            <div className="form-group">
              <label>Options</label>
              {currentQuestion.options?.map((option, index) => (
                <input
                  key={index}
                  type="text"
                  value={option}
                  onChange={(e) => updateOption(index, e.target.value)}
                  placeholder={`Option ${index + 1}`}
                  className="input"
                />
              ))}
            </div>
          )}

          {currentQuestion.type === 'true_false' && (
            <div className="form-group">
              <label htmlFor="tf-answer">Correct Answer</label>
              <select
                id="tf-answer"
                value={currentQuestion.correct_answer}
                onChange={(e) => setCurrentQuestion({ ...currentQuestion, correct_answer: e.target.value })}
                className="input"
              >
                <option value="">Select answer</option>
                <option value="true">True</option>
                <option value="false">False</option>
              </select>
            </div>
          )}

          {(currentQuestion.type === 'mcq' || currentQuestion.type === 'short_answer' || currentQuestion.type === 'fill_blank') && (
            <div className="form-group">
              <label htmlFor="correct-answer">Correct Answer</label>
              <input
                id="correct-answer"
                type="text"
                value={currentQuestion.correct_answer}
                onChange={(e) => setCurrentQuestion({ ...currentQuestion, correct_answer: e.target.value })}
                placeholder="Enter correct answer"
                className="input"
              />
            </div>
          )}

          <div className="form-group">
            <label htmlFor="explanation">Explanation (optional)</label>
            <textarea
              id="explanation"
              value={currentQuestion.explanation}
              onChange={(e) => setCurrentQuestion({ ...currentQuestion, explanation: e.target.value })}
              placeholder="Explain the correct answer"
              className="input textarea"
              rows={2}
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="difficulty">Difficulty</label>
              <select
                id="difficulty"
                value={currentQuestion.difficulty}
                onChange={(e) => setCurrentQuestion({ ...currentQuestion, difficulty: e.target.value })}
                className="input"
              >
                <option value="easy">Easy</option>
                <option value="medium">Medium</option>
                <option value="hard">Hard</option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="points">Points</label>
              <input
                id="points"
                type="number"
                min="1"
                max="10"
                value={currentQuestion.points}
                onChange={(e) => setCurrentQuestion({ ...currentQuestion, points: parseInt(e.target.value) })}
                className="input"
              />
            </div>
          </div>

          <button onClick={addQuestion} className="button button-secondary">
            Add Question
          </button>
        </div>

        <div className="form-actions">
          <button
            onClick={saveQuiz}
            disabled={saving || questions.length === 0}
            className="button button-primary button-large"
          >
            {saving ? 'Saving...' : 'Save Quiz'}
          </button>
        </div>
      </div>
    </div>
  );
}