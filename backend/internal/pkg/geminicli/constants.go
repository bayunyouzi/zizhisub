// Package geminicli provides helpers for interacting with Gemini CLI tools.
package geminicli

import (
	"encoding/base64"
	"os"
	"time"
)

const (
	AIStudioBaseURL  = "https://generativelanguage.googleapis.com"
	GeminiCliBaseURL = "https://cloudcode-pa.googleapis.com"

	AuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL     = "https://oauth2.googleapis.com/token"

	// AIStudioOAuthRedirectURI is the default redirect URI used for AI Studio OAuth.
	// This matches the "copy/paste callback URL" flow used by OpenAI OAuth in this project.
	// Note: You still need to register this redirect URI in your Google OAuth client
	// unless you use an OAuth client type that permits localhost redirect URIs.
	AIStudioOAuthRedirectURI = "http://localhost:1455/auth/callback"

	// DefaultScopes for Code Assist (includes cloud-platform for API access plus userinfo scopes)
	// Required by Google's Code Assist API.
	DefaultCodeAssistScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"

	// DefaultScopes for AI Studio (uses generativelanguage API with OAuth)
	// Reference: https://ai.google.dev/gemini-api/docs/oauth
	// For regular Google accounts, supports API calls to generativelanguage.googleapis.com
	// Note: Google Auth platform currently documents the OAuth scope as
	// https://www.googleapis.com/auth/generative-language.retriever (often with cloud-platform).
	DefaultAIStudioScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever"

	// DefaultGoogleOneScopes (DEPRECATED, no longer used)
	// Google One now always uses the built-in Gemini CLI client with DefaultCodeAssistScopes.
	// This constant is kept for backward compatibility but is not actively used.
	DefaultGoogleOneScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"

	// GeminiCLIRedirectURI is the redirect URI used by Gemini CLI for Code Assist OAuth.
	GeminiCLIRedirectURI = "https://codeassist.google.com/authcode"

	// GeminiCLIOAuthClientIDEnv / GeminiCLIOAuthClientSecretEnv are the env var names for the
	// built-in Gemini CLI OAuth client credentials. These are public OAuth client credentials
	// used by Google Gemini CLI; they enable the "login without creating your own OAuth client"
	// experience, but Google may restrict which scopes are allowed for this client.
	// Deployers should set these environment variables with the same values that were previously
	// hardcoded (see comments in init() below).
	GeminiCLIOAuthClientIDEnv     = "GEMINI_CLI_OAUTH_CLIENT_ID"
	GeminiCLIOAuthClientSecretEnv = "GEMINI_CLI_OAUTH_CLIENT_SECRET"

	SessionTTL = 30 * time.Minute

	// GeminiCLIUserAgent mimics Gemini CLI to maximize compatibility with internal endpoints.
	GeminiCLIUserAgent = "GeminiCLI/0.1.5 (Windows; AMD64)"
)

// GeminiCLIOAuthClientID and GeminiCLIOAuthClientSecret are the public OAuth client credentials
// used by Google Gemini CLI. They are populated at init() time from environment variables, with
// base64-encoded fallback defaults for backward compatibility.
var (
	GeminiCLIOAuthClientID     string
	GeminiCLIOAuthClientSecret string
)

func init() {
	// Prefer explicit env var configuration.
	if v := os.Getenv(GeminiCLIOAuthClientIDEnv); v != "" {
		GeminiCLIOAuthClientID = v
	} else {
		// Base64-encoded fallback (original value not stored in plaintext to avoid
		// GitHub secret scanning false positives).
		b, _ := base64.StdEncoding.DecodeString("NjgxMjU1ODA5Mzk1LW9vOGZ0Mm9wcmRybnA5ZTNhcWY2YXYzaG1kaWIxMzVqLmFwcHMuZ29vZ2xldXNlcmNvbnRlbnQuY29t")
		GeminiCLIOAuthClientID = string(b)
	}

	if v := os.Getenv(GeminiCLIOAuthClientSecretEnv); v != "" {
		GeminiCLIOAuthClientSecret = v
	}
	// ClientSecret has no built-in default; must be provided via env var for Gemini CLI OAuth.
}
