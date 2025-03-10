package auth

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"slices"
)

const (
	RoleAdmin = "admin"
	RoleDev   = "ngic-dev"
	RoleUser  = "user"
)

var ROLES = []string{RoleAdmin, RoleDev, RoleUser}

type Settings struct {
	Realm             string `json:"realm"`
	BaseURL           string `json:"baseURL"`
	ClientID          string `json:"clientID"`
	ClientSecret      string `json:"clientSecret"`
	RedirectURI       string `json:"redirectURI"`
	OAuthStateCookie  string `json:"oauthStateCookie" default:"oauthstate"`
	StateCookieMaxAge int    `json:"stateCookieMaxAge" default:"300"`
	JWTKey            []byte `json:"jwtKey"`
}

type Claims struct {
	Username  string   `json:"username"`
	Role      string   `json:"role,omitempty"`
	Namespace []string `json:"namespaces,omitempty"`
	jwt.RegisteredClaims
}

type KeycloakClaim struct {
	PreferredUsername string `json:"preferred_username"`
	// realm_access.roles is typically an array of roles assigned to the user.
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	// Optionally, custom claims such as allowed_namespaces can be added.
	AllowedNamespaces []string `json:"allowed_namespaces,omitempty"`
	jwt.RegisteredClaims
}

// TokenResponse represents the JSON response from Keycloak's token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func GetSettings() (*Settings, error) {
	var s Settings
	err := envconfig.Process("deprestart", &s)
	return &s, err
}

func (settings *Settings) GetAuthURL() string {
	return fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/auth", settings.BaseURL, url.PathEscape(settings.Realm))
}

func (settings *Settings) GetTokenURL() string {
	return fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", settings.BaseURL, url.PathEscape(settings.Realm))
}

func exchangeCodeForToken(code string) (*TokenResponse, error) {
	settings, err := GetSettings()
	if err != nil {
		return nil, err
	}
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", settings.ClientID)
	data.Set("client_secret", settings.ClientSecret)
	data.Set("redirect_uri", settings.RedirectURI)

	resp, err := http.PostForm(settings.GetTokenURL(), data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func parseIDToken(idToken string) (*KeycloakClaim, error) {
	parser := new(jwt.Parser)
	claims := &KeycloakClaim{}

	// Parse without verifying the signature
	_, _, err := parser.ParseUnverified(idToken, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func determineRole(roles []string) string {
	for _, r := range ROLES {
		if slices.Contains(roles, r) {
			return r
		}
	}

	return RoleUser
}

func generateState(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
