import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

interface Quiz {
  id: string;
  title: string;
  description?: string;
  questions: any[];
  created_at: string;
}

export function Dashboard() {
  const [quizzes, setQuizzes] = useState<Quiz[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    totalQuizzes: 0,
    totalQuestions: 0,
    avgScore: 0
  });

  useEffect(() => {
    fetchQuizzes();
  }, []);

  const fetchQuizzes = async () => {
    try {
      const response = await fetch('/api/v1/quizzes?limit=10');
      const data = await response.json();
      setQuizzes(data.quizzes || []);

      // Calculate stats
      const total = data.quizzes?.length || 0;
      const questions = data.quizzes?.reduce((sum: number, q: Quiz) =>
        sum + (q.questions?.length || 0), 0) || 0;

      setStats({
        totalQuizzes: total,
        totalQuestions: questions,
        avgScore: 0 // This would come from results API
      });
    } catch (error) {
      console.error('Failed to fetch quizzes:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading dashboard...</div>;
  }

  return (
    <div className="dashboard">
      <div className="page-header">
        <h1>Dashboard</h1>
        <div className="quick-actions">
          <Link to="/generate" className="button button-primary">
            🤖 Generate Quiz
          </Link>
          <Link to="/create" className="button button-secondary">
            ✏️ Create Quiz
          </Link>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{stats.totalQuizzes}</div>
          <div className="stat-label">Total Quizzes</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.totalQuestions}</div>
          <div className="stat-label">Total Questions</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.avgScore}%</div>
          <div className="stat-label">Average Score</div>
        </div>
      </div>

      <div className="recent-quizzes">
        <h2>Recent Quizzes</h2>
        {quizzes.length === 0 ? (
          <div className="empty-state">
            <p>No quizzes yet. Create your first quiz!</p>
            <Link to="/generate" className="button button-primary">
              Get Started
            </Link>
          </div>
        ) : (
          <div className="quiz-list">
            {quizzes.map((quiz) => (
              <div key={quiz.id} className="quiz-card">
                <h3>{quiz.title}</h3>
                {quiz.description && <p>{quiz.description}</p>}
                <div className="quiz-meta">
                  <span>{quiz.questions?.length || 0} questions</span>
                  <span>{new Date(quiz.created_at).toLocaleDateString()}</span>
                </div>
                <div className="quiz-actions">
                  <Link to={`/quiz/${quiz.id}`} className="button button-small">
                    Take Quiz
                  </Link>
                  <Link to={`/analytics?quiz=${quiz.id}`} className="button button-small button-outline">
                    View Results
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}