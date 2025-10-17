import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'react-hot-toast';
import { Header } from './components/Header';
import { Dashboard } from './pages/Dashboard';
import { ScenarioAnalysis } from './pages/ScenarioAnalysis';
import { ProgramGeneration } from './pages/ProgramGeneration';
import { Implementation } from './pages/Implementation';
import { Analytics } from './pages/Analytics';
import './App.css';

// Create React Query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <div className="app">
          <Header />
          <main className="main-content">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/analyze" element={<ScenarioAnalysis />} />
              <Route path="/generate" element={<ProgramGeneration />} />
              <Route path="/implement" element={<Implementation />} />
              <Route path="/analytics" element={<Analytics />} />
            </Routes>
          </main>
          <Toaster
            position="top-right"
            toastOptions={{
              duration: 4000,
              style: {
                background: '#363636',
                color: '#fff',
              },
            }}
          />
        </div>
      </Router>
    </QueryClientProvider>
  );
}

export default App;