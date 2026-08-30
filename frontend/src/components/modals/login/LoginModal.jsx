import React, { useState } from 'react';
import ReactDOM from 'react-dom';
import { useHistory } from 'react-router-dom';
import { PersonInput, LockInput } from '../../inputs/InputBar';
import SiteButton from '../../buttons/SiteButtons';
import { useAuth } from '../../../helpers/AuthContent';

const LoginModal = ({ isOpen, onClose, onLogin, redirectAfterLogin }) => {
    const { signup } = useAuth();
    const [isSignUp, setIsSignUp] = useState(false);
    const [username, setUsername] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const history = useHistory();

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
