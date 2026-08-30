const getEnvValue = (key, fallback) => {
  if (typeof window === 'undefined' || !window.__ENV__) return fallback;
  const val = window.__ENV__[key];
  if (!val || val.startsWith('${') || val.startsWith('__')) return fallback;
  return val;
};

export const DOMAIN_URL = getEnvValue('DOMAIN_URL', typeof window !== 'undefined' ? window.location.origin : 'http://localhost');
export const API_URL = getEnvValue('API_URL', typeof window !== 'undefined' ? window.location.origin : 'http://localhost');
