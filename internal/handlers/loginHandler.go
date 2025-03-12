package handlers

import (
	"deployment-restarter/pkg/auth"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/url"
	"time"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a random state string and store it in a cookie.
	settings, _ := auth.GetSettings()
	state := auth.GenerateState(16)
	http.SetCookie(w, &http.Cookie{
		Name:   settings.OAuthStateCookie,
		Value:  state,
		MaxAge: settings.StateCookieMaxAge,
	})

	// Build the Keycloak authorization URL.
	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=%s",
		settings.GetAuthURL(),
		url.QueryEscape(settings.ClientID),
		url.QueryEscape(settings.RedirectURI),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	settings, _ := auth.GetSettings()
	// Verify state.
	queryState := r.URL.Query().Get("state")
	cookie, err := r.Cookie(settings.OAuthStateCookie)
	if err != nil || cookie.Value != queryState {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for tokens.
	tokenResp, err := auth.ExchangeCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange error: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse the id_token to extract user claims.
	kcClaims, err := auth.ParseIDToken(tokenResp.IDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse id_token: %v", err), http.StatusInternalServerError)
		return
	}

	// Determine the user role based on Keycloak roles.
	role := auth.DetermineRole(kcClaims.RealmAccess.Roles)

	// Create our own JWT token to track the session.
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &auth.Claims{
		Username:  kcClaims.PreferredUsername,
		Role:      role,
		Namespace: nil,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{Time: expirationTime},
		},
	}
	ourToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := ourToken.SignedString(settings.JWTKey)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// Set our JWT in a cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
	// Redirect to the main page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
