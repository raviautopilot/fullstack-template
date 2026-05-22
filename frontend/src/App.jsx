import React, { useState, useEffect } from 'react';

const BACKEND_URL = 'http://localhost:8080';

function App() {
  const [token, setToken] = useState(null);
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [health, setHealth] = useState(null);
  const [healthLoading, setHealthLoading] = useState(true);
  const [healthError, setHealthError] = useState(false);

  // Parse token from URL query or localStorage at startup
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const urlToken = urlParams.get('token');
    
    if (urlToken) {
      localStorage.setItem('auth_token', urlToken);
      setToken(urlToken);
      // Clean url parameters
      window.history.replaceState({}, document.title, window.location.pathname);
    } else {
      const storedToken = localStorage.getItem('auth_token');
      if (storedToken) {
        setToken(storedToken);
      } else {
        setLoading(false);
      }
    }
  }, []);

  // Fetch user profile when token is set
  useEffect(() => {
    if (!token) {
      setUser(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    fetch(`${BACKEND_URL}/api/user`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
      .then(res => {
        if (!res.ok) {
          throw new Error('Unauthorized');
        }
        return res.json();
      })
      .then(data => {
        setUser(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Failed to fetch user:', err);
        // Clean corrupted token
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
        setLoading(false);
      });
  }, [token]);

  // Fetch health check status (polling every 5 seconds)
  const fetchHealth = () => {
    setHealthLoading(true);
    fetch(`${BACKEND_URL}/health`)
      .then(res => {
        if (!res.ok) {
          throw new Error('Server down');
        }
        return res.json();
      })
      .then(data => {
        setHealth(data);
        setHealthError(false);
        setHealthLoading(false);
      })
      .catch(err => {
        console.error('Health check failed:', err);
        setHealth(null);
        setHealthError(true);
        setHealthLoading(false);
      });
  };

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleLogin = () => {
    // Redirect to Go API OAuth login endpoint
    window.location.href = `${BACKEND_URL}/auth/login`;
  };

  const handleLogout = () => {
    fetch(`${BACKEND_URL}/auth/logout`)
      .then(() => {
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
      })
      .catch(err => {
        console.error('Logout error:', err);
        // Force logout on error
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
      });
  };

  if (loading) {
    return (
      <div className="auth-wrapper">
        <div style={{ textAlign: 'center' }}>
          <div className="auth-logo-group">
            <div className="auth-logo-dot"></div>
            <div className="auth-logo-dot"></div>
          </div>
          <p style={{ marginTop: '16px', color: 'var(--color-text-secondary)' }}>Loading environment secure sessions...</p>
        </div>
      </div>
    );
  }

  // ----------------------------------------------------
  // LOGGED OUT VIEW
  // ----------------------------------------------------
  if (!user) {
    return (
      <>
        <div className="auth-wrapper">
          <div className="glass-card auth-card" data-testid="login-card">
            <div className="auth-logo-group">
              <div className="auth-logo-dot"></div>
              <div className="auth-logo-dot"></div>
              <div className="auth-logo-dot"></div>
              <div className="auth-logo-dot"></div>
            </div>
            <h1 className="auth-title">Antigravity Console</h1>
            <p className="auth-subtitle">
              Access the system control panel. Monitor live health metrics, runtime telemetry, and manage authentication configurations.
            </p>

            <button 
              className="btn-primary" 
              data-testid="google-login-btn"
              onClick={handleLogin}
            >
              <svg className="google-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12.24 10.285V13.4h6.887C18.2 15.614 15.645 18 12.24 18c-3.86 0-7-3.14-7-7s3.14-7 7-7c1.7 0 3.3.6 4.5 1.7l2.4-2.4C17.3 1.5 14.9.7 12.24.7 6.54.7 1.94 5.3 1.94 11s4.6 10.3 10.3 10.3c5.9 0 9.8-4.1 9.8-10 0-.67-.06-1.3-.16-2.015H12.24z"/>
              </svg>
              Sign in with Google
            </button>

            {/* Health status indicator during login */}
            <div 
              style={{ marginTop: '36px', borderTop: '1px solid var(--border-color)', paddingTop: '24px', textAlign: 'left' }}
              data-testid="health-status-card"
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '14px', fontWeight: '500', color: 'var(--color-text-secondary)' }}>Server Telemetry Status</span>
                {healthError ? (
                  <span className="badge badge-offline" data-testid="health-status-badge">
                    <span className="pulse"></span> Offline
                  </span>
                ) : healthLoading && !health ? (
                  <span style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>Polling...</span>
                ) : (
                  <span className="badge badge-online" data-testid="health-status-badge">
                    <span className="pulse"></span> Online
                  </span>
                )}
              </div>
              {health && (
                <div style={{ display: 'flex', gap: '20px', marginTop: '14px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
                  <div>Env: <strong style={{ color: 'var(--color-text-secondary)' }}>{health.environment}</strong></div>
                  <div>Uptime: <strong style={{ color: 'var(--color-text-secondary)' }}>{Math.floor(health.uptime_seconds)}s</strong></div>
                  <div>Mem: <strong style={{ color: 'var(--color-text-secondary)' }}>{health.memory.alloc_mb} MB</strong></div>
                </div>
              )}
            </div>
          </div>
        </div>
        <footer className="footer">
          &copy; {new Date().getFullYear()} Antigravity Cloud Inc. All rights reserved.
        </footer>
      </>
    );
  }

  // ----------------------------------------------------
  // LOGGED IN DASHBOARD VIEW
  // ----------------------------------------------------
  return (
    <>
      <header className="header" data-testid="dashboard-header">
        <div className="brand">
          <div style={{ width: '10px', height: '10px', borderRadius: '50%', background: 'var(--gradient-main)' }}></div>
          Antigravity Control Console
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <span style={{ fontSize: '14px', color: 'var(--color-text-secondary)' }}>
            Welcome, <strong>{user.name}</strong>
          </span>
          <button className="btn-secondary" onClick={handleLogout}>Sign out</button>
        </div>
      </header>

      <div className="main-container">
        <div className="dashboard-grid">
          {/* HEALTH MONITORING METRICS */}
          <div className="glass-card" data-testid="health-status-card">
            <div className="health-status-header">
              <h2 className="section-title" style={{ marginBottom: 0 }}>System Telemetry & Health</h2>
              {healthError ? (
                <span className="badge badge-offline" data-testid="health-status-badge">
                  <span className="pulse"></span> Offline
                </span>
              ) : health?.dependencies?.google_oauth?.status === 'DEGRADED' ? (
                <span className="badge badge-degraded" data-testid="health-status-badge">
                  <span className="pulse"></span> Degraded
                </span>
              ) : (
                <span className="badge badge-online" data-testid="health-status-badge">
                  <span className="pulse"></span> Online
                </span>
              )}
            </div>

            {health ? (
              <>
                <div className="grid-metrics">
                  <div className="metric-card">
                    <div className="metric-label">Environment</div>
                    <div className="metric-value" style={{ textTransform: 'capitalize', color: 'var(--accent-primary)' }}>
                      {health.environment}
                    </div>
                  </div>
                  <div className="metric-card">
                    <div className="metric-label">System Uptime</div>
                    <div className="metric-value">
                      {Math.floor(health.uptime_seconds)}s
                    </div>
                  </div>
                  <div className="metric-card">
                    <div className="metric-label">Memory Allocated</div>
                    <div className="metric-value">
                      {health.memory.alloc_mb} MB
                    </div>
                  </div>
                </div>

                <h3 style={{ fontSize: '15px', fontWeight: '600', marginBottom: '16px', color: 'var(--color-text-secondary)' }}>Memory Allocation Breakdown</h3>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '32px' }}>
                  <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', marginBottom: '6px' }}>
                      <span>Active Heap Size (Alloc)</span>
                      <span>{health.memory.alloc_mb} MB / {health.memory.sys_mb} MB</span>
                    </div>
                    <div style={{ width: '100%', height: '8px', background: 'rgba(255,255,255,0.05)', borderRadius: '4px', overflow: 'hidden' }}>
                      <div style={{ 
                        height: '100%', 
                        background: 'var(--gradient-main)', 
                        width: `${Math.min(100, (health.memory.alloc_mb / health.memory.sys_mb) * 100)}%`,
                        transition: 'width 0.5s' 
                      }}></div>
                    </div>
                  </div>
                  <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', marginBottom: '6px' }}>
                      <span>Cumulative Allocated (Total Alloc)</span>
                      <span>{health.memory.total_alloc_mb} MB</span>
                    </div>
                    <div style={{ width: '100%', height: '8px', background: 'rgba(255,255,255,0.05)', borderRadius: '4px', overflow: 'hidden' }}>
                      <div style={{ 
                        height: '100%', 
                        background: 'var(--accent-secondary)', 
                        width: '45%' 
                      }}></div>
                    </div>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', color: 'var(--color-text-muted)' }}>
                    <span>Garbage Collector (GC) Cycles Running</span>
                    <span>{health.memory.num_gc} runs</span>
                  </div>
                </div>

                <h3 style={{ fontSize: '15px', fontWeight: '600', marginBottom: '16px', color: 'var(--color-text-secondary)' }}>Dependency Check status</h3>
                <div className="metric-card" style={{ borderLeft: '3px solid var(--accent-success)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span style={{ fontSize: '14px', fontWeight: '600' }}>Google OAuth 2.0 Integration</span>
                    <span style={{ 
                      fontSize: '12px', 
                      fontWeight: '600', 
                      color: health.dependencies.google_oauth.status === 'UP' ? 'var(--accent-success)' : 'var(--accent-warning)'
                    }}>
                      {health.dependencies.google_oauth.status}
                    </span>
                  </div>
                  <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginTop: '6px' }}>
                    {health.dependencies.google_oauth.details}
                  </p>
                </div>
              </>
            ) : (
              <div style={{ textAlign: 'center', padding: '40px 0', color: 'var(--accent-danger)' }}>
                <span style={{ fontSize: '32px' }}>⚠️</span>
                <p style={{ marginTop: '12px', fontWeight: '500' }}>Unable to contact the backend service. Check server status.</p>
              </div>
            )}
          </div>

          {/* LOGGED IN USER PROFILE */}
          <div className="glass-card user-profile" data-testid="user-profile-card">
            <h2 className="section-title" style={{ width: '100%', textAlign: 'left' }}>User Session</h2>
            {user.picture ? (
              <img className="user-avatar" src={user.picture} alt={user.name} />
            ) : (
              <div className="user-avatar-placeholder">{user.name.charAt(0)}</div>
            )}
            <h3 className="user-name">{user.name}</h3>
            <p className="user-email">{user.email}</p>

            <div className="user-meta-group">
              <div className="meta-row">
                <span className="meta-label">Authority Type</span>
                <span className="meta-value" style={{ color: 'var(--accent-success)' }}>Administrator</span>
              </div>
              <div className="meta-row">
                <span className="meta-label">Session Status</span>
                <span className="meta-value">Active JWT</span>
              </div>
              <div className="meta-row" style={{ flexDirection: 'column', gap: '6px', marginTop: '12px' }}>
                <span className="meta-label" style={{ marginBottom: '2px' }}>Encoded JWT Token</span>
                <textarea 
                  readOnly 
                  value={token}
                  style={{
                    width: '100%',
                    height: '60px',
                    background: 'rgba(0,0,0,0.25)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '8px',
                    color: 'var(--color-text-muted)',
                    fontSize: '11px',
                    padding: '8px',
                    fontFamily: 'monospace',
                    resize: 'none',
                    outline: 'none'
                  }}
                />
              </div>
            </div>

            <button 
              className="btn-primary" 
              style={{ background: 'linear-gradient(135deg, #ef4444, #f97316)', boxShadow: '0 4px 12px rgba(239, 68, 68, 0.2)' }}
              data-testid="logout-btn"
              onClick={handleLogout}
            >
              Terminate Session
            </button>
          </div>
        </div>
      </div>

      <footer className="footer">
        &copy; {new Date().getFullYear()} Antigravity Cloud Inc. All rights reserved.
      </footer>
    </>
  );
}

export default App;
