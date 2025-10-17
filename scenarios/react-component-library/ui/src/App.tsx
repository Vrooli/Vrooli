import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from 'react-query';
import { Toaster } from 'react-hot-toast';

// Layout
import Layout from './components/Layout';

// Pages
import Dashboard from './pages/Dashboard';
import ComponentLibrary from './pages/ComponentLibrary';
import ComponentDetails from './pages/ComponentDetails';
import CreateComponent from './pages/CreateComponent';
import Testing from './pages/Testing';
import Analytics from './pages/Analytics';
import AIGenerator from './pages/AIGenerator';

// Styles
import './styles/App.css';

// Create a client for react-query
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
        <div className="App">
          <Layout>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/library" element={<ComponentLibrary />} />
              <Route path="/component/:id" element={<ComponentDetails />} />
              <Route path="/create" element={<CreateComponent />} />
              <Route path="/generate" element={<AIGenerator />} />
              <Route path="/testing" element={<Testing />} />
              <Route path="/analytics" element={<Analytics />} />
            </Routes>
          </Layout>
          <Toaster
            position="top-right"
            toastOptions={{
              duration: 4000,
              style: {
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #374151',
              },
            }}
          />
        </div>
      </Router>
    </QueryClientProvider>
  );
}

export default App;