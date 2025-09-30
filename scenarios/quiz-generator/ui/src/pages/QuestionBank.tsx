import React, { useState } from 'react';

export function QuestionBank() {
  const [searchQuery, setSearchQuery] = useState('');
  const [questions, setQuestions] = useState([]);

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;

    try {
      const response = await fetch('/api/v1/question-bank/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: searchQuery, limit: 20 })
      });
      const data = await response.json();
      setQuestions(data.questions || []);
    } catch (error) {
      console.error('Search failed:', error);
    }
  };

  return (
    <div className="question-bank">
      <h1>Question Bank</h1>
      
      <div className="search-bar">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search questions..."
          className="input"
          onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button onClick={handleSearch} className="button button-primary">
          Search
        </button>
      </div>

      <div className="questions-list">
        {questions.map((q: any, i) => (
          <div key={i} className="question-item">
            <p>{q.question}</p>
            <div className="question-meta">
              <span>{q.type}</span>
              <span>{q.difficulty}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}