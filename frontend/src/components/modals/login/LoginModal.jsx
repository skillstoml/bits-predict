import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import { useHistory } from 'react-router-dom';
import { PersonInput, LockInput } from '../../inputs/InputBar';
import SiteButton from '../../buttons/SiteButtons';
import { useAuth } from '../../../helpers/AuthContent';
import { GOOGLE_CLIENT_ID } from '../../../config';

const LoginModal = ({ isOpen, onClose, onLogin, redirectAfterLogin }) => {
    const { signup, loginWithGoogle } = useAuth();
    const [isSignUp, setIsSignUp] = useState(false);
    const [username, setUsername] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const history = useHistory();

    const handleGoogleLoginSuccess = async (response) => {
        setError('');
        try {
            const loginResult = await loginWithGoogle(response.credential);
            if (loginResult?.success) {
                onClose();
                history.push(getPostLoginDestination(loginResult.mustChangePassword));
            }
        } catch (authError) {
            console.error('Google Auth error:', authError);
            setError(authError.message || 'Google authentication failed.');
        }
    };

    useEffect(() => {
        if (!isOpen) return;

        const id = 'google-gsi-client';
        let script = document.getElementById(id);
        if (!script) {
            script = document.createElement('script');
            script.src = 'https://accounts.google.com/gsi/client';
            script.id = id;
            script.async = true;
            script.defer = true;
            document.body.appendChild(script);
        }

        const initializeGoogleSignIn = () => {
            if (!GOOGLE_CLIENT_ID) {
                console.warn("GOOGLE_CLIENT_ID is not configured in client environment");
                return;
            }

            if (window.google) {
                window.google.accounts.id.initialize({
                    client_id: GOOGLE_CLIENT_ID,
                    callback: handleGoogleLoginSuccess,
                });
                window.google.accounts.id.renderButton(
                    document.getElementById('google-signin-btn'),
                    { theme: 'outline', size: 'large', width: 272 }
                );
            }
        };

        script.onload = () => {
            initializeGoogleSignIn();
        };

        if (window.google) {
            initializeGoogleSignIn();
        }
    }, [isOpen]);

    const getPostLoginDestination = (mustChangePassword) => {
        if (mustChangePassword) {
            return '/changepassword';
        }

        const fallbackPath = '/markets';
        const safeRedirects = new Set([
            '/',
            '/about',
            '/markets',
            '/polls',
            '/stats',
            '/style',
        ]);

        return safeRedirects.has(redirectAfterLogin) ? redirectAfterLogin : fallbackPath;
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');

        try {
            if (isSignUp) {
                const domainSuffix = "bits-pilani.ac.in";
                const trimmedEmail = email.trim().toLowerCase();
                if (!trimmedEmail.endsWith(domainSuffix)) {
                    setError("Only BITS Pilani email addresses are allowed (@*bits-pilani.ac.in)");
                    return;
                }

                const loginResult = await signup(username.trim().toLowerCase(), trimmedEmail, password);
                if (loginResult?.success) {
                    onClose();
                    history.push(getPostLoginDestination(loginResult.mustChangePassword));
                }
            } else {
                const loginResult = await onLogin(username.trim(), password);
                if (loginResult?.success) {
                    onClose();
                    history.push(getPostLoginDestination(loginResult.mustChangePassword));
                }
            }
        } catch (authError) {
            console.error('Authentication error:', authError);
            setError(authError.message || 'An error occurred. Please try again.');
        }
    };

    if (!isOpen) return null;

    return ReactDOM.createPortal(
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex justify-center items-center">
            <div className="relative bg-blue-900 p-6 rounded-lg text-white max-w-sm mx-auto w-80">
                <h2 className="text-xl mb-4">{isSignUp ? 'Sign Up' : 'Login'}</h2>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <PersonInput value={username} onChange={(e) => {
                        setUsername(e.target.value);
                    }} />

                    {isSignUp && (
                        <div className="flex items-center border-2 border-blue-500 bg-transparent rounded-md">
                            <span className="h-5 w-5 text-blue-500 ml-2">✉️</span>
                            <input
                                type="email"
                                placeholder="BITS Email (@*bits-pilani.ac.in)"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                className="flex-1 px-4 py-2 rounded-md text-white bg-transparent focus:outline-none"
                                required
                            />
                        </div>
                    )}

                    <LockInput value={password} onChange={(e) => {
                        setPassword(e.target.value);
                    }} />
                    {error && <div className='error-message text-red-400 text-sm mt-1'>{error}</div>}
                    <div className="flex flex-col gap-3 mt-4">
                        <SiteButton type="submit">{isSignUp ? 'Sign Up' : 'Login'}</SiteButton>
                        
                        {GOOGLE_CLIENT_ID && (
                            <>
                                <div className="flex items-center my-1">
                                    <hr className="flex-1 border-gray-700" />
                                    <span className="px-2 text-[10px] text-gray-400 font-semibold uppercase tracking-wider">OR</span>
                                    <hr className="flex-1 border-gray-700" />
                                </div>
                                <div className="w-full flex justify-center mb-1">
                                    <div id="google-signin-btn"></div>
                                </div>
                            </>
                        )}

                        <button
                            type="button"
                            onClick={() => {
                                setIsSignUp(!isSignUp);
                                setError('');
                            }}
                            className="text-xs text-blue-400 hover:text-blue-300 text-left mt-1 underline"
                        >
                            {isSignUp ? 'Already have an account? Login' : "Don't have an account? Sign Up"}
                        </button>
                    </div>
                </form>
                <button className="absolute top-0 right-0 mt-4 mr-4 text-gray-400 hover:text-white" onClick={onClose}>
                    ✕
                </button>
            </div>
        </div>,
        document.getElementById('modal-root')
    );
};

export default LoginModal;
